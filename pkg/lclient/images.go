package lclient

import (
	"bytes"
	"io"
	"os"

	"github.com/docker/docker/api/types"
	"github.com/golang/glog"
	"github.com/pkg/errors"
)

// PullImage uses Docker to pull the specified image. If there are any issues pulling the image,
// it returns an error.
func (dc *Client) PullImage(image string) error {

	out, err := dc.Client.ImagePull(dc.Context, image, types.ImagePullOptions{})
	defer out.Close()

	if err != nil {
		return errors.Wrapf(err, "Unable to pull image")
	}

	if glog.V(4) {
		_, err := io.Copy(os.Stdout, out)
		if err != nil {
			return err
		}
	} else {
		// Need to read from the buffer or else Docker won't finish pulling the image
		buf := new(bytes.Buffer)
		_, err := buf.ReadFrom(out)
		if err != nil {
			return err
		}
	}

	return nil
}
