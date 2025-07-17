package cmd

func InitCommands() {
	RootCmd.AddCommand(GetCmd)
	RootCmd.AddCommand(setCmd)
}
