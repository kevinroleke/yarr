package main

import (
	"fmt"
	"time"
	"context"
	"strings"
	"strconv"
	"errors"
	"crypto/sha1"
	"html/template"

	"github.com/mmcdole/gofeed"
)

func Hash(input string) string {
	h := sha1.New()
	h.Write([]byte(input))
	bs := h.Sum(nil)

	return fmt.Sprintf("%x", bs)[0:32]
}

func parsePubDate(pubDate string) (time.Time, error) {
	// Wed, 15 Sep 2021 00:00:00 -0400

	months := map[string]time.Month{
		"Jan": 1,
		"Feb": 2,
		"Mar": 3,
		"Apr": 4,
		"May": 5,
		"Jun": 6,
		"Jul": 7,
		"Aug": 8,
		"Sep": 9,
		"Oct": 10,
		"Nov": 11,
		"Dec": 12,
	}

	var (
		year int
		day int
		hour int
		min int
		sec int
	)

	comp := strings.Split(pubDate, " ")
	month := months[comp[2]]
	tim := strings.Split(comp[4], ":")
	nsec := 0
	location := time.UTC

	strArgs := map[*int]string {
		&year: comp[3],
		&day: comp[1],
		&hour: tim[0],
		&min: tim[1],
		&sec: tim[2],
	}
	
	for k, v := range strArgs {
		i, err := strconv.Atoi(v)
		if err != nil {
			return time.Time{}, err
		}
		*k = i
	}

	return time.Date(year, month, day, hour, min, sec, nsec, location), nil
}

func UpdateRss(rssLink string) error {
	ctx, cancel := context.WithTimeout(context.Background(), 8*time.Second)
	defer cancel()
	fp := gofeed.NewParser()
	feed, _ := fp.ParseURLWithContext(rssLink, ctx)
	
	// fmt.Printf("%s\n\n%s\n\n%s\n\nBy %s\n\n%s\n%s", feed.Title, feed.Description, feed.Image.URL, feed.Author.Name, feed.Categories, feed.Categories)

	id := Hash(rssLink)
	exists, err := PodExists(id)
	if err != nil {
		return err
	}

	if feed == nil {
		return errors.New("Feed is nil")
	}

	var pImage string
	var author string
	if feed.Image == nil {
		pImage = "/static/logo.jpeg"
	} else {
		pImage = feed.Image.URL
	}
	if feed.Author == nil {
		author = "Unknown"
	} else {
		author = feed.Author.Name
	}

	if !exists {
		pod := Pod{
			Title: feed.Title,
			Description: template.HTML(Ps.Sanitize(feed.Description)),
			AlbumArt: pImage,
			Creator: author,
			Categories: feed.Categories,
			Rss: rssLink,
			Id: id,
			Added: time.Now(),
			Link: feed.Link,
		}

		AddPod(pod)
	}

	episodes, err := GetAllEpisodes(id)
	if err != nil {
		return err
	}

	for _, item := range feed.Items {
		eid := Hash(rssLink + item.Title)
		stop := false
		for _, episode := range episodes {
			if eid == episode.Id {
				stop = true
			}
		}

		if stop || item == nil {
			continue
		}

		pubDate, err := parsePubDate(item.Published)
		if err != nil {
			pubDate = time.Now()
		}

		if len(item.Enclosures) == 0 {
			return errors.New("no content")
		}

		var image string
		if item.Image == nil {
			image = pImage
		} else {
			image = item.Image.URL
		}

		episode := Episode{
			Title: item.Title,
			Description: template.HTML(Ps.Sanitize(item.Description)),
			Media: item.Enclosures[0].URL,
			MediaType: item.Enclosures[0].Type,
			Thumbnail: image,
			Id: eid,
			Published: pubDate,
			PodId: id,
		}
		fmt.Printf("[*] Added episode `%s`\n", episode.Title)
		AddEpisode(episode)
	}

	return nil
}