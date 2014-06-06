package main

import (
	"errors"
	"io/ioutil"
	"os"
	"path/filepath"
	"sort"
	"strconv"
	"strings"

	"github.com/disintegration/imaging"
	"github.com/rwcarlsen/goexif/exif"
)

func getAlbums(group string) ([]string, error) {
	groupPath := filepath.Join(root, group)
	f, err := os.Open(groupPath)
	if err != nil {
		return nil, err
	}
	return f.Readdirnames(-1)
}

type sortFileInfo []os.FileInfo

func (s sortFileInfo) Len() int           { return len(s) }
func (s sortFileInfo) Swap(i, j int)      { s[i], s[j] = s[j], s[i] }
func (s sortFileInfo) Less(i, j int) bool { return s[i].ModTime().Before(s[j].ModTime()) }

func getImages(group, album string) (string, []string, error) {
	albumPath := filepath.Join(root, group, album)
	f, err := os.Open(albumPath)
	if err != nil {
		return "", nil, err
	}
	files, err := f.Readdir(-1)
	if err != nil {
		return "", nil, err
	}
	sort.Sort(sortFileInfo(files))
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
		lname := strings.ToLower(name)

		isImage := false
		isImage = isImage || strings.HasSuffix(lname, ".jpg")
		isImage = isImage || strings.HasSuffix(lname, ".jpeg")
		isImage = isImage || strings.HasSuffix(lname, ".png")
		if !isImage {
			continue
		}
		images = append(images, name)
	}
	// sort.StringSlice(images).Sort()
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
	err = os.MkdirAll(filepath.Join(root, group, album, cacheDir), 0777)
	if err != nil {
		return "", err
	}
	cachePath := filepath.Join(root, group, album, cacheDir, cacheImageName)
	cacheImage, err := os.Open(cachePath)
	if err != nil {
		if !os.IsNotExist(err) {
			return "", err
		}
		// Resize image, open cache image.
		fullImagePath := filepath.Join(root, group, album, image)
		f, err := os.Open(fullImagePath)
		if err != nil {
			return "", err
		}
		meta, err := exif.Decode(f)
		f.Close()
		rotateImage := 0
		if err == nil {
			tag, err := meta.Get(exif.Orientation)
			if err == nil && tag.Count > 0 {
				rotateImage = int(tag.Int(0))
			}
		}
		fullImage, err := imaging.Open(fullImagePath)
		if err != nil {
			return "", err
		}

		switch rotateImage {
		case 0:
			// No rotation.
		case 1:
			// No rotation.
		case 2:
			// Left-right flip.
			fullImage = imaging.FlipV(fullImage)
		case 3:
			// Rot 180 deg.
			fullImage = imaging.Rotate180(fullImage)
		case 4:
			// Top-bottom flip.
			fullImage = imaging.FlipH(fullImage)
		case 5:
			// Rot 90 deg, left-right flip.
			fullImage = imaging.Rotate90(fullImage)
			fullImage = imaging.FlipV(fullImage)
		case 6:
			// Rot 270 deg.
			fullImage = imaging.Rotate270(fullImage)
		case 7:
			// Rot 90 deg, top-bottom flip.
			fullImage = imaging.Rotate90(fullImage)
			fullImage = imaging.FlipH(fullImage)
		case 8:
			// Rot 90 deg.
			fullImage = imaging.Rotate90(fullImage)
		default:
			log.Warning("Unknown exif orientation value: %d", rotateImage)
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
