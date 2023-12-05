package shell

type IExec interface {
	Run(cmdStr string) (string, error)
	RunFile(cmdStr string) (string, error)
	Close() error
}
