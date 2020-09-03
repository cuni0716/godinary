package interactors

import (
	"fmt"
	raven "github.com/getsentry/raven-go"
	"godinary/image"
	"godinary/storage"
	"log"
	"time"
)

func createJobFromImageURL(imageURL string) (*image.Job, error) {
	job := image.NewJob()
	job.AcceptWebp = false
	err := job.Parse(imageURL, true)
	return job, err
}

func storeImageAndPrintLogs(job *image.Job, body []byte, storageDriver storage.Driver, logger *log.Logger, existTime float64, downloadTime float64) {
	currentTime := time.Now()
	err := storageDriver.Write(body, job.Source.Hash, "source/")
	storageTime := time.Since(currentTime).Seconds()
	totalTime := existTime + downloadTime + storageTime
	timeString := fmt.Sprintf("\n\t=> TOTAL %0.5fs - CACHE %0.5fs - DOWNLOAD %0.5fs - STORE %0.5fs",
		totalTime, existTime, downloadTime, storageTime)
	if err != nil {
		logger.Printf("Error: Image could not be stored: \"%s\" with error %s"+timeString, job.Source.URL, err)
		raven.CaptureErrorAndWait(err, nil)
	} else {
		logger.Printf("Image saved correctly: %s"+timeString, job.Source.URL)
	}
}

//DownloadAndCacheImage This function cache a given url in our storage. If the given url_image
// is already cached doesn't do nothing and return nil, if we couldn't douwnload the image return nil
func DownloadAndCacheImage(imageURL string, storageDriver storage.Driver, async bool, logger *log.Logger) error {
	job, err := createJobFromImageURL(imageURL)
	if err != nil {
		logger.Printf("Error: URL could not be parsed: %s", imageURL)
		return err
	}

	currentTime := time.Now()
	exist, err := storageDriver.Exists(job.Source.Hash, "source/")
	existTime := time.Since(currentTime).Seconds()
	if exist == true {
		logger.Printf("Image URL is already cached: %s\n\t=> CACHE %0.5fs", imageURL, existTime)
	} else if err != nil {
		logger.Printf("Error: Could not acces to google cloud storage with image url: \"%s\". Error %s", imageURL, err)
		raven.CaptureErrorAndWait(err, nil)
		return err
	} else {
		currentTime := time.Now()
		body, err := job.Source.Download()
		downloadTime := time.Since(currentTime).Seconds()
		if err != nil {
			logger.Printf("Image could not be downloaded: %s \n\t=> CACHE %0.5fs - DOWNLOAD %0.5fs",
				imageURL, existTime, downloadTime)
			return nil
		}
		if async {
			go func() {
				raven.CapturePanic(func() {
					storeImageAndPrintLogs(job, body, storageDriver, logger, existTime, downloadTime)
				}, nil)
			}()
		} else {
			storeImageAndPrintLogs(job, body, storageDriver, logger, existTime, downloadTime)
		}
	}
	return nil
}
