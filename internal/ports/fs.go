package ports

type Filesystem interface {
	Save(path string, data []byte) error
}

type NoopFilesystem struct{}

func (NoopFilesystem) Save(_ string, _ []byte) error { return nil }

var _ Filesystem = NoopFilesystem{}
