package oci

import (
	"archive/tar"
	"bytes"
	"fmt"
	"io"
	"os"
	"path/filepath"
	"time"

	"github.com/google/go-containerregistry/pkg/crane"
	"github.com/google/go-containerregistry/pkg/name"
	v1 "github.com/google/go-containerregistry/pkg/v1"
	"github.com/google/go-containerregistry/pkg/v1/mutate"
	"github.com/google/go-containerregistry/pkg/v1/tarball"
	log "github.com/rjbrown57/binman/pkg/logging"
)

type BinmanImageBuild struct {
	Assets       []string // list of files that should be in the final image
	BaseImage    string   // The list of baseimages to stack and add our layer to. Nil signals the creation of a scratch image
	Name         string
	Registry     string
	ImageBinPath string // where assets will be written
	Version      string

	layers []v1.Layer
	image  v1.Image
}

func MakeBinmanImageBuild(targetImageName, imagePath, baseImage string) (BinmanImageBuild, error) {

	r, err := name.NewTag(targetImageName)
	if err != nil {
		return BinmanImageBuild{}, err
	}

	return BinmanImageBuild{
		Registry:     r.RegistryStr(),
		Version:      r.TagStr(),
		Name:         r.RepositoryStr(),
		ImageBinPath: imagePath,
		BaseImage:    baseImage,
	}, nil
}

// https://gist.github.com/ahmetb/430baa4e8bb0b0f78abb1c34934cd0b6
func (bib *BinmanImageBuild) getLayers() error {

	for _, file := range bib.Assets {
		log.Tracef("reading %s", file)
		target := fmt.Sprintf("%s/%s", bib.ImageBinPath, filepath.Base(file))

		f, err := os.Stat(file)
		if err != nil {
			return err
		}

		fileBytes, err := os.ReadFile(file)
		if err != nil {
			log.Warnf("Unable to read %s - %s", file, err)
			return err
		}

		// The below is a based on crane.Layer
		// https://github.com/google/go-containerregistry/blob/main/pkg/crane/filemap.go#L31
		b := new(bytes.Buffer)
		layerTar := tar.NewWriter(b)

		tarHeader := &tar.Header{
			Name:    target,
			Mode:    int64(f.Mode()),
			Size:    f.Size(), // size of file in int64
			ModTime: f.ModTime(),
		}

		err = layerTar.WriteHeader(tarHeader)
		if err != nil {
			return err
		}

		_, err = layerTar.Write(fileBytes)
		if err != nil {
			return err
		}

		if err := layerTar.Close(); err != nil {
			return err
		}

		l, _ := tarball.LayerFromOpener(func() (io.ReadCloser, error) {
			return io.NopCloser(bytes.NewBuffer(b.Bytes())), nil
		})

		bib.layers = append(bib.layers, l)
	}

	return nil
}

func (bib *BinmanImageBuild) makeImage() error {

	switch bib.BaseImage {
	case "", "scratch":
		log.Fatalf("scratch images not currently supported")
		return nil
	default:
		base, err := crane.Pull(bib.BaseImage)
		if err != nil {
			log.Warnf("Issue pulling base image %s", err)
			return err
		}

		var layerstoadd []mutate.Addendum

		for i, l := range bib.layers {
			layerstoadd = append(layerstoadd, mutate.Addendum{Layer: l,
				History: v1.History{EmptyLayer: false,
					CreatedBy: fmt.Sprintf("%s", filepath.Base(bib.Assets[i])),
					Comment:   "Built with rjbrown57/binman",
					Created:   v1.Time{Time: time.Now()}}})
		}

		bib.image, err = mutate.Append(base, layerstoadd...)
	}
	return nil
}

// BuildOciImage will build an OCI compatible image
func BuildOciImage(bib *BinmanImageBuild) error {

	var err error

	err = bib.getLayers()
	if err != nil {
		return err
	}

	err = bib.makeImage()
	if err != nil {
		return err
	}

	imageName := fmt.Sprintf("%s/%s:%s", bib.Registry, bib.Name, bib.Version)

	log.Debugf("Writing image %s", imageName)

	err = crane.Push(bib.image, imageName)
	if err != nil {
		log.Warnf("Error pushing image %s", err)
	}

	s, err := crane.Digest(imageName)
	log.Debugf("image %s written. Digest = %s", imageName, s)

	return nil
}
