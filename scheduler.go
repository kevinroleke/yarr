package main

import (
	"fmt"
	"time"
)

// go EveryHour(UpdateAll)

func EveryHour(cb func()) {
	for {
		go cb()

		time.Sleep(1 * time.Hour)
	}
}

func UpdateAll() {
	fmt.Println("[*] Updating all feeds")
	pods, err := GetAllApprovedPods()
	if err != nil {
		fmt.Println("Unable to GetAllApprovedPods")
		fmt.Println(err)
		return
	}

	for _, pod := range pods {
		UpdateRss(pod.Rss)
	}
}