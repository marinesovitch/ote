// Darek Slusarczyk alias marines marinesovitch 2021, 2022

package setup

import (
	"encoding/json"
	"errors"
	"fmt"
	"os"
)

type ImageItem int

const (
	ShellImage ImageItem = iota
	ShellCommercialImage
	RouterImage
	ServerImage
	ImageItemLast
)

func ParseImageItem(itemStr string) (ImageItem, error) {
	switch itemStr {
	case "shell":
		return ShellImage, nil

	case "shell-commercial":
		return ShellCommercialImage, nil

	case "router":
		return RouterImage, nil

	case "server":
		return ServerImage, nil

	default:
		return -1, errors.New("unknown image item " + itemStr)
	}
}

type ImageInfo struct {
	Item string
	Name string
	Id   string
	Pull bool
}

type Images struct {
	images []ImageInfo
}

func (i *Images) Load(path string) error {
	imageJson, err := os.ReadFile(path)
	if err != nil {
		return err
	}

	var images []ImageInfo
	err = json.Unmarshal(imageJson, &images)
	if err != nil {
		return err
	}

	if i.images == nil {
		i.images = make([]ImageInfo, ImageItemLast)
	}

	for _, imageInfo := range images {
		imageItem, err := ParseImageItem(imageInfo.Item)
		if err != nil {
			return err
		}
		i.images[imageItem] = imageInfo
	}

	return nil
}

func (i *Images) GetImageInfo(item ImageItem) (ImageInfo, error) {
	if len(i.images) < int(item) {
		return ImageInfo{}, errors.New("images not loaded")
	}

	imageInfo := i.images[item]
	if len(imageInfo.Name) == 0 {
		return imageInfo, fmt.Errorf("image '%s' not loaded", imageInfo.Item)
	}

	return imageInfo, nil
}

func (i *Images) GetName(item ImageItem) {
}
