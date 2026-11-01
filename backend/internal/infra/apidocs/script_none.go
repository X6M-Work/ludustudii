//go:build !apidocs

package apidocs

// readScript returns ErrBuiltWithoutDocs: builds without the apidocs tag leave
// the Scalar bundle out of the binary.
func readScript() ([]byte, error) {
	return nil, ErrBuiltWithoutDocs
}
