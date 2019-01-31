package importer

import (
	"errors"
	"os"
	"path/filepath"
	"strings"

	. "github.com/onsi/ginkgo"
	"github.com/onsi/ginkgo/extensions/table"
	. "github.com/onsi/gomega"

	cdiv1 "kubevirt.io/containerized-data-importer/pkg/apis/core/v1alpha1"
	"kubevirt.io/containerized-data-importer/pkg/controller"
	"kubevirt.io/containerized-data-importer/pkg/image"
	"kubevirt.io/containerized-data-importer/pkg/util"
)

var (
	imageFile            = filepath.Join(imageDir, "registry-image.tar")
	invalidImageFile     = filepath.Join(imageDir, "docker-image.tar")
	imageData            = filepath.Join(imageDir, "data")
	tmpData              = filepath.Join(imageData, "tmp")
	diskImage            = filepath.Join(imageData, "disk.img")
	validImageURL        = "docker://image.url"
	invalidDestIndicator = "invalid"
	invalidImageURL      = "docker://" + invalidDestIndicator
)

type fakeSkopeoOperations struct {
	e1 error
}

var _ = Describe("Copy from Registry", func() {

	BeforeEach(func() {
		By("[BeforeEach] Creating working directory")
		os.Mkdir(imageData, os.ModeDir|os.ModePerm)
	})

	AfterEach(func() {
		By("[AfterEach]  deleting working directory")
		os.RemoveAll(imageData)
	})

	table.DescribeTable("Image, with import source should", func(destImage string, dataDir string, url string, skopeoOperations image.SkopeoOperations, wantErr bool) {
		defer os.RemoveAll(destImage)
		By("Replacing Skopeo Operations")
		replaceSkopeoOperations(skopeoOperations, func() {
			By("Copying image")
			err := CopyData(&DataStreamOptions{
				destImage,
				dataDir,
				url,
				"",
				"",
				controller.SourceRegistry,
				string(cdiv1.DataVolumeKubeVirt),
				"1G"})
			if !wantErr {
				Expect(err).NotTo(HaveOccurred())
			} else {
				Expect(err).To(HaveOccurred())
			}
		})
	},
		table.Entry("successfully copy registry image", diskImage, imageData, validImageURL, NewFakeSkopeoOperations(nil), false),
		table.Entry("expect failure trying to copy non-existing image", diskImage, "fake", validImageURL, NewSkopeoAllErrors(), true),
		table.Entry("expect failure trying to copy invalid image", diskImage, imageData, invalidImageURL, NewSkopeoAllErrors(), true),
	)
})

func replaceSkopeoOperations(replacement image.SkopeoOperations, f func()) {
	orig := image.SkopeoInterface
	if replacement != nil {
		image.SkopeoInterface = replacement
		defer func() { image.SkopeoInterface = orig }()
	}
	f()
}

func NewSkopeoAllErrors() image.SkopeoOperations {
	err := errors.New("skopeo should not be called from this test override with replaceSkopeoOperations")
	return NewFakeSkopeoOperations(err)
}

func NewFakeSkopeoOperations(e1 error) image.SkopeoOperations {
	return &fakeSkopeoOperations{e1}
}

func (o *fakeSkopeoOperations) CopyImage(url, dest, accessKey, secKey string) error {
	if o.e1 == nil {
		if strings.Contains(url, invalidDestIndicator) {
			if err := util.UnArchiveLocalTar(invalidImageFile, tmpData); err != nil {
				return errors.New("could not extract layer tar")
			}
		} else {
			if err := util.UnArchiveLocalTar(imageFile, tmpData); err != nil {
				return errors.New("could not extract layer tar")
			}
		}
	}
	return o.e1
}
