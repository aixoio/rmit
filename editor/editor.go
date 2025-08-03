package editor

import (
	"context"
	"encoding/json"
	"fmt"
	"os"
	"os/exec"
	"strings"

	"github.com/charmbracelet/huh"
	"github.com/charmbracelet/huh/spinner"
	"github.com/charmbracelet/lipgloss"
	"github.com/openai/openai-go"
	"github.com/openai/openai-go/option"
	"github.com/spf13/viper"
)

func gitStatus() (string, error) {
	cmd := exec.Command("git", "status", "--porcelain", "-b")
	out, err := cmd.CombinedOutput()
	if err != nil {
		return "", fmt.Errorf("git status failed: %w", err)
	}
	return string(out), nil
}

// gitTrackedFiles returns all files in the current working directory tracked by git
func gitTrackedFiles() (string, error) {
	cmd := exec.Command("git", "ls-files")
	out, err := cmd.CombinedOutput()
	if err != nil {
		return "", fmt.Errorf("git ls-files failed: %w", err)
	}
	return string(out), nil
}

func gitDiff() (string, error) {
	staged := exec.Command("git", "diff", "--staged")
	so, _ := staged.Output()
	if len(so) > 0 {
		return string(so), nil
	}
	unstaged := exec.Command("git", "diff")
	uo, err := unstaged.Output()
	if err != nil {
		return "", fmt.Errorf("git diff failed: %w", err)
	}
	return string(uo), nil
}

func readFile(path string) (string, error) {
	b, err := os.ReadFile(path)
	if err != nil {
		return "", err
	}
	return string(b), nil
}

func llm(client *openai.Client, model string, messages []openai.ChatCompletionMessageParamUnion, tools []openai.ChatCompletionToolParam) (*openai.ChatCompletion, error) {
	cc, err := client.Chat.Completions.New(context.TODO(), openai.ChatCompletionNewParams{
		Messages: messages,
		Model:    model,
		Tools:    tools,
	})
	if err != nil {
		return nil, err
	}
	return cc, nil
}

func StartEditor(initial string) (string, error) {
	// Model selection:
	// - chatModel is used for the interactive chat loop
	// - defaultModel is used ONLY for the final commit generation
	defaultModel := viper.GetString("default_model")
	if strings.TrimSpace(defaultModel) == "" {
		if m := os.Getenv("RMIT_DEFAULT_MODEL"); strings.TrimSpace(m) != "" {
			defaultModel = m
		} else {
			defaultModel = string(openai.ChatModelGPT4o)
		}
	}
	chatModel := viper.GetString("chat_model")
	if strings.TrimSpace(chatModel) == "" {
		if m := os.Getenv("RMIT_CHAT_MODEL"); strings.TrimSpace(m) != "" {
			chatModel = m
		} else {
			chatModel = defaultModel
		}
	}

	// BaseURL/API key: prefer viper, then env; avoid empty BaseURL (causes unsupported protocol scheme "")
	baseURL := viper.GetString("api_url")
	if strings.TrimSpace(baseURL) == "" {
		baseURL = os.Getenv("RMIT_API_URL")
	}
	apiKey := viper.GetString("api_key")
	if strings.TrimSpace(apiKey) == "" {
		apiKey = os.Getenv("RMIT_API_KEY")
	}

	client := openai.NewClient(
		option.WithBaseURL(baseURL),
		option.WithAPIKey(apiKey),
	)

	// Lipgloss styles for chat + tool usage (improved UI)
	title := lipgloss.NewStyle().Bold(true).Foreground(lipgloss.Color("99")).Render
	sep := lipgloss.NewStyle().Foreground(lipgloss.Color("60")).Render
	box := lipgloss.NewStyle().
		Padding(0, 1).
		Border(lipgloss.RoundedBorder()).
		BorderForeground(lipgloss.Color("60")).
		Margin(0, 0, 1, 0)

	assistantStyle := lipgloss.NewStyle().
		Foreground(lipgloss.Color("15")). // white text
		Background(lipgloss.Color("57")). // purple-ish bg
		Padding(0, 1).
		Align(lipgloss.Left)

	userStyle := lipgloss.NewStyle().
		Foreground(lipgloss.Color("0")).  // black text
		Background(lipgloss.Color("14")). // teal-ish bg
		Padding(0, 1).
		Align(lipgloss.Left)

	roleAssistant := lipgloss.NewStyle().Bold(true).Foreground(lipgloss.Color("177")).Render
	roleUser := lipgloss.NewStyle().Bold(true).Foreground(lipgloss.Color("51")).Render

	toolStyle := lipgloss.NewStyle().
		Foreground(lipgloss.Color("8")).
		MarginLeft(2)

	// Define tools per v1.12.0 docs: FunctionDefinitionParam + FunctionParameters
	tools := []openai.ChatCompletionToolParam{
		{
			Function: openai.FunctionDefinitionParam{
				Name:        "gitStatus",
				Description: openai.String("Get concise git status with branch and porcelain output"),
				Parameters: openai.FunctionParameters{
					"type":       "object",
					"properties": map[string]any{},
					"required":   []string{},
				},
			},
		},
		{
			Function: openai.FunctionDefinitionParam{
				Name:        "gitDiff",
				Description: openai.String("Get the current diff (staged if present, otherwise unstaged)"),
				Parameters: openai.FunctionParameters{
					"type":       "object",
					"properties": map[string]any{},
					"required":   []string{},
				},
			},
		},
		{
			Function: openai.FunctionDefinitionParam{
				Name:        "readFile",
				Description: openai.String("Read a text file from disk"),
				Parameters: openai.FunctionParameters{
					"type": "object",
					"properties": map[string]any{
						"path": map[string]any{
							"type":        "string",
							"description": "Absolute or relative file path to read",
						},
					},
					"required": []string{"path"},
				},
			},
		},
		{
			Function: openai.FunctionDefinitionParam{
				Name:        "gitTrackedFiles",
				Description: openai.String("List all files in the current working directory that are tracked by git"),
				Parameters: openai.FunctionParameters{
					"type":       "object",
					"properties": map[string]any{},
					"required":   []string{},
				},
			},
		},
	}

	system := "You are a commit message editor. Keep replies concise. Use tools to gather context, and have a brief back-and-forth with the user before finalizing. " +
		"Strive to ask at least one clarifying question if helpful. When you are ready to output the final commit message, respond only with: FINAL: <commit message>"

	messages := []openai.ChatCompletionMessageParamUnion{
		openai.SystemMessage(system),
		openai.UserMessage("Seed message based on diff: \n" + strings.TrimSpace(initial)),
	}

	// Intro header
	fmt.Println(box.Render(title("Commit Message Editor")))
	fmt.Println(sep("Enter details below. The assistant will ask clarifying questions if needed.\n"))

	// Keep a visible running transcript. We'll re-render last exchange for clarity.
	// Note: openai-go message params hold a Role field and a Content union. We only ever send simple text,
	// so we can safely get content by formatting the param.
	// contentToString extracts plain text from message param unions without dumping Go structs
	contentToString := func(m openai.ChatCompletionMessageParamUnion) string {
		switch mm := any(m).(type) {
		case openai.ChatCompletionUserMessageParam:
			return strings.TrimSpace(fmt.Sprintf("%s", mm.Content))
		case openai.ChatCompletionAssistantMessageParam:
			return strings.TrimSpace(fmt.Sprintf("%s", mm.Content))
		case openai.ChatCompletionSystemMessageParam:
			return ""
		default:
			// Avoid raw struct dumps
			return ""
		}
	}

	history := func() string {
		var b strings.Builder
		for _, m := range messages {
			switch any(m).(type) {
			case openai.ChatCompletionUserMessageParam:
				txt := contentToString(m)
				if txt != "" {
					b.WriteString(roleUser("You") + ": " + userStyle.Render(txt) + "\n")
				}
			case openai.ChatCompletionAssistantMessageParam:
				txt := contentToString(m)
				if txt != "" {
					b.WriteString(roleAssistant("Assistant") + ": " + assistantStyle.Render(txt) + "\n")
				}
			default:
				// skip system/others to avoid raw prints
			}
		}
		return b.String()
	}

	for {
		var cc *openai.ChatCompletion
		var callErr error
		_ = spinner.New().Title("Thinking...").Action(func() {
			var e error
			cc, e = llm(&client, chatModel, messages, tools)
			if e != nil {
				callErr = e
			}
		}).Run()
		if callErr != nil {
			return "", fmt.Errorf("LLM error: %w", callErr)
		}

		choice := cc.Choices[0]
		assistantMsg := choice.Message

		assistantText := strings.TrimSpace(assistantMsg.Content)

		// Stop if FINAL - detect before normal assistant print to prevent duplication
		if strings.HasPrefix(assistantText, "FINAL:") {
			final := strings.TrimSpace(strings.TrimPrefix(assistantText, "FINAL:"))

			// Ask the defaultModel to produce just the final commit message (idempotent step)
			confirmMsgs := []openai.ChatCompletionMessageParamUnion{
				openai.SystemMessage(system),
				openai.UserMessage("Finalize the commit message. Respond only with: FINAL: <commit message>."),
				openai.AssistantMessage("FINAL: " + final),
			}
			var cc2 *openai.ChatCompletion
			var err2 error
			_ = spinner.New().Title("Finalizing with default model...").Action(func() {
				cc2, err2 = llm(&client, defaultModel, confirmMsgs, nil)
			}).Run()
			output := final
			if err2 == nil && cc2 != nil {
				if t := strings.TrimSpace(cc2.Choices[0].Message.Content); strings.HasPrefix(t, "FINAL:") {
					output = strings.TrimSpace(strings.TrimPrefix(t, "FINAL:"))
				}
			}

			fmt.Println(roleAssistant("Final Commit Message") + ":")
			fmt.Println(box.Render(assistantStyle.Render(output)))
			return output, nil
		}

		// Normal assistant print (only if not FINAL)
		if assistantText != "" {
			fmt.Println(roleAssistant("Assistant") + ":")
			fmt.Println(box.Render(assistantStyle.Render(assistantText)))
		}

		// Maintain conversation using helper
		messages = append(messages, assistantMsg.ToParam())

		// Handle tool calls
		if len(assistantMsg.ToolCalls) > 0 {
			for _, tc := range assistantMsg.ToolCalls {
				name := tc.Function.Name
				argsJSON := tc.Function.Arguments
				var toolOutput string
				var toolErr error

				switch name {
				case "gitStatus":
					toolOutput, toolErr = gitStatus()
				case "gitDiff":
					toolOutput, toolErr = gitDiff()
				case "readFile":
					var params struct {
						Path string `json:"path"`
					}
					if err := json.Unmarshal([]byte(argsJSON), &params); err != nil {
						toolErr = fmt.Errorf("invalid arguments for readFile: %w", err)
						break
					}
					toolOutput, toolErr = readFile(strings.TrimSpace(params.Path))
				case "gitTrackedFiles":
					toolOutput, toolErr = gitTrackedFiles()
				default:
					toolErr = fmt.Errorf("unknown tool: %s", name)
				}

				// Show tool usage in gray, small indent; DO NOT print output
				toolLabel := name
				if name == "readFile" {
					var pm map[string]any
					_ = json.Unmarshal([]byte(argsJSON), &pm)
					if p, ok := pm["path"].(string); ok && strings.TrimSpace(p) != "" {
						toolLabel += " (" + strings.TrimSpace(p) + ")"
					}
				}
				// Tool usage should be silent by default; gate behind verbose flag
				if viper.GetBool("verbose_tools") || strings.EqualFold(os.Getenv("RMIT_VERBOSE_TOOLS"), "true") {
					fmt.Println(toolStyle.Render("Using tool: " + toolLabel))
				}

				if toolErr != nil {
					// Inform model of error via tool message
					messages = append(messages, openai.ToolMessage("ERROR: "+toolErr.Error(), tc.ID))
					continue
				}

				// Append tool result back to the model without printing it
				messages = append(messages, openai.ToolMessage(strings.TrimSpace(toolOutput), tc.ID))
			}
			// Loop to let the model consume tool results
			continue
		}

		// Ask user if no tool calls and not FINAL, while showing history
		fmt.Println(sep(strings.Repeat("─", 40)))
		fmt.Print(history())

		readerMsg := ""
		// Use a simple, non-styled title and a placeholder to avoid huh layout issues
		_ = huh.NewForm(
			huh.NewGroup(
				huh.NewInput().
					Title("Your reply").
					Placeholder("Type your response and press Enter").
					Value(&readerMsg),
			),
		).Run()
		if strings.TrimSpace(readerMsg) == "" {
			continue
		}
		fmt.Println(roleUser("You") + ":")
		fmt.Println(box.Render(userStyle.Render(readerMsg)))
		messages = append(messages, openai.UserMessage(readerMsg))
	}
}
