package main

import (
	"errors"
	"io/ioutil"
	"os"
	"path/filepath"
	"strconv"
	"strings"

	"github.com/disintegration/imaging"
)

func getAlbums(group string) ([]string, error) {
	groupPath := filepath.Join(*root, group)
	f, err := os.Open(groupPath)
	if err != nil {
		return nil, err
	}
	return f.Readdirnames(-1)
}

func getImages(group, album string) (string, []string, error) {
	albumPath := filepath.Join(*root, group, album)
	f, err := os.Open(albumPath)
	if err != nil {
		return "", nil, err
	}
	files, err := f.Readdir(-1)
	if err != nil {
		return "", nil, err
	}
	description := ""
	images := make([]string, 0, len(files)-1)
	for _, fi := range files {
		if fi.IsDir() {
			continue
		}
		name := fi.Name()
		if name[0] == '.' {
			continue
		}
		if name == descriptionFile {
			bb, err := ioutil.ReadFile(filepath.Join(albumPath, name))
			if err != nil {
				return "", nil, err
			}
			description = string(bb)
		}

		isImage := false
		isImage = isImage || strings.HasSuffix(name, ".jpg")
		isImage = isImage || strings.HasSuffix(name, ".jpeg")
		isImage = isImage || strings.HasSuffix(name, ".png")
		if !isImage {
			continue
		}
		images = append(images, name)
	}
	return description, images, nil
}

var badImageSize = errors.New("Bad image size")

func getSingleImage(group, album, res, image string) (string, error) {
	size, err := strconv.Atoi(res)
	if err != nil {
		return "", err
	}
	imgSize := 0
	for _, sz := range sizes {
		if sz == size {
			imgSize = sz
			break
		}
	}
	if imgSize <= 0 {
		return "", badImageSize
	}

	ext := filepath.Ext(image)
	cacheImageName := image[:len(image)-len(ext)] + "@" + res + ext
	err = os.MkdirAll(filepath.Join(*root, group, album, cacheDir), 0777)
	if err != nil {
		return "", err
	}
	cachePath := filepath.Join(*root, group, album, cacheDir, cacheImageName)
	cacheImage, err := os.Open(cachePath)
	if err != nil {
		if !os.IsNotExist(err) {
			return "", err
		}
		// Resize image, open cache image.
		fullImagePath := filepath.Join(*root, group, album, image)
		fullImage, err := imaging.Open(fullImagePath)
		if err != nil {
			return "", err
		}
		resized := imaging.Fit(fullImage, imgSize, imgSize, imaging.Linear)
		err = imaging.Save(resized, cachePath)
		if err != nil {
			return "", err
		}
	}
	cacheImage.Close()
	return cachePath, nil
}
