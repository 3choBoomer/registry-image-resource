package main

import (
	"encoding/json"
	"fmt"
	"os"
	"path/filepath"

	resource "github.com/concourse/registry-image-resource"
	color "github.com/fatih/color"
	"github.com/google/go-containerregistry/pkg/name"
	"github.com/google/go-containerregistry/pkg/v1/remote"
	"github.com/sirupsen/logrus"
)

type InRequest struct {
	Source  resource.Source  `json:"source"`
	Version resource.Version `json:"version"`
}

type InResponse struct {
	Version  resource.Version         `json:"version"`
	Metadata []resource.MetadataField `json:"metadata"`
}

type ImageMetadata struct {
	Env  []string `json:"env"`
	User string   `json:"user"`
}

func main() {
	logrus.SetOutput(os.Stderr)
	logrus.SetFormatter(&logrus.TextFormatter{
		ForceColors: true,
	})

	color.NoColor = false

	var req InRequest
	err := json.NewDecoder(os.Stdin).Decode(&req)
	if err != nil {
		logrus.Errorf("invalid payload: %s", err)
		os.Exit(1)
		return
	}

	if req.Source.Debug {
		logrus.SetLevel(logrus.DebugLevel)
	}

	if len(os.Args) < 2 {
		logrus.Errorf("destination path not specified")
		os.Exit(1)
		return
	}

	dest := os.Args[1]

	ref := req.Source.Repository + "@" + req.Version.Digest

	n, err := name.ParseReference(ref, name.WeakValidation)
	if err != nil {
		logrus.Errorf("failed to resolve name: %s", err)
		os.Exit(1)
		return
	}

	fmt.Fprintf(os.Stderr, "fetching %s@%s\n", color.GreenString(req.Source.Repository), color.YellowString(req.Version.Digest))

	image, err := remote.Image(n)
	if err != nil {
		logrus.Errorf("failed to locate remote image: %s", err)
		os.Exit(1)
		return
	}

	err = unpackImage(filepath.Join(dest, "rootfs"), image, req.Source.Debug)
	if err != nil {
		logrus.Errorf("failed to extract image: %s", err)
		os.Exit(1)
		return
	}

	cfg, err := image.ConfigFile()
	if err != nil {
		logrus.Errorf("failed to inspect image config: %s", err)
		os.Exit(1)
		return
	}

	meta, err := os.Create(filepath.Join(dest, "metadata.json"))
	if err != nil {
		logrus.Errorf("failed to create image metadata: %s", err)
		os.Exit(1)
		return
	}

	err = json.NewEncoder(meta).Encode(ImageMetadata{
		Env:  cfg.ContainerConfig.Env,
		User: cfg.ContainerConfig.User,
	})
	if err != nil {
		logrus.Errorf("failed to write image metadata: %s", err)
		os.Exit(1)
		return
	}

	err = meta.Close()
	if err != nil {
		logrus.Errorf("failed to close image metadata file: %s", err)
		os.Exit(1)
		return
	}

	json.NewEncoder(os.Stdout).Encode(InResponse{
		Version:  req.Version,
		Metadata: []resource.MetadataField{},
	})
}
