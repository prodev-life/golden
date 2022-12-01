package git

// $ gitversion=$(git describe); go build -ldflags "-X golden/pkg/git.Version=${gitversion}"`
const noVersionMessage = `Version is not initialized.`

var Version = noVersionMessage
