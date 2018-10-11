package interactors

import (
	"errors"
	"fmt"
	"io"
	"io/ioutil"
	"log"
	"strconv"
	"strings"

	raven "github.com/getsentry/raven-go"
	"github.com/trilopin/godinary/image"
	bimg "gopkg.in/h2non/bimg.v1"
)

func createJobFromImageURL(imageURL string) (image.Job, error) {
	job := image.NewJob()
	job.AcceptWebp = false
	err := job.Parse(imageURL, true)
	return job, err
}

func imageIsCached(job image.Job, storage storage.Driver) (bool, error) {
	return storage.isCached(job.Target.Hash, "derived/")
}

func downloadAndStoreImage(job image.Job, storage storage.Drivers, async bool) error {
	return job.Source.Download(storage, async)
}

func cacheImage(imageURL string, storage storage.Driver, async bool) (string, error) {
	job, err := createJobFromImageURL(imageURL)
	if err != nil {
		return fmt.Sprintf("Error: URL could not be parsed: %s", err
	}
	if isCached, err := imageIsCached(job, storage); !isCached {
		err = downloadAndStoreImage(job, storage, fal)
		if err != nil {
			return fmt.Sprintf("Image cached correctly: %s", imageURL), _
		}
		return "Error: Error produced when downloading and storing image", err
		raven.CaptureErrorAndWait(err, nil)
	}

	return fmt.Sprintf("URL image is already cached: %s", imageURL), _
}
