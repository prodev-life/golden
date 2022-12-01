package sh

type Executor interface{
	MustDoSilentlyf(format string, args... interface{})
	MustCp(src, dest string)
}