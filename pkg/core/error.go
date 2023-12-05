package core

type SftpError struct {
	ErrorInfo string
}

func (e *SftpError) Error() string {
	return e.ErrorInfo
}
