package helpers

import (
	"KMS/devices"
	"encoding/json"
	"errors"
	"fmt"
	"log"
	"math"
	"math/rand"
	"os"
	"os/exec"
	"strconv"
	"time"
)

type F func(devices.UBNTDevices) int

func Timer(ubnts devices.UBNTDevices, seconds time.Duration) {
	ticker := time.NewTicker(2 * time.Second)

	for _ = range ticker.C {
		ticker := time.NewTicker(seconds * time.Second)

		for _ = range ticker.C {
			fmt.Println("tock")
			b, err := json.MarshalIndent(ubnts, "", "  ")
			if err != nil {
				fmt.Println("error:", err)
			}
			os.Stdout.Write(b)

			for i, _ := range ubnts {
				fmt.Printf("%#v\n", ubnts[i].PDUs[3])
			}
		}
	}
}

func TimerPrint(str string, seconds time.Duration) {
	ticker := time.NewTicker(2 * time.Second)

	for _ = range ticker.C {
		ticker := time.NewTicker(seconds * time.Second)

		for _ = range ticker.C {
			fmt.Println("\r\b" + str)

		}
	}
}

func plural(count int, singular string) (result string) {
	if (count == 1) || (count == 0) {
		result = strconv.Itoa(count) + " " + singular + " "
	} else {
		result = strconv.Itoa(count) + " " + singular + "s "
	}
	return
}

func secondsToHuman(input int) (result string) {
	years := math.Floor(float64(input) / 60 / 60 / 24 / 7 / 30 / 12)
	seconds := input % (60 * 60 * 24 * 7 * 30 * 12)
	months := math.Floor(float64(seconds) / 60 / 60 / 24 / 7 / 30)
	seconds = input % (60 * 60 * 24 * 7 * 30)
	weeks := math.Floor(float64(seconds) / 60 / 60 / 24 / 7)
	seconds = input % (60 * 60 * 24 * 7)
	days := math.Floor(float64(seconds) / 60 / 60 / 24)
	seconds = input % (60 * 60 * 24)
	hours := math.Floor(float64(seconds) / 60 / 60)
	seconds = input % (60 * 60)
	minutes := math.Floor(float64(seconds) / 60)
	seconds = input % 60

	if years > 0 {
		result = plural(int(years), "year") + plural(int(months), "month") + plural(int(weeks), "week") + plural(int(days), "day") + plural(int(hours), "hour") + plural(int(minutes), "minute") + plural(int(seconds), "second")
	} else if months > 0 {
		result = plural(int(months), "month") + plural(int(weeks), "week") + plural(int(days), "day") + plural(int(hours), "hour") + plural(int(minutes), "minute") + plural(int(seconds), "second")
	} else if weeks > 0 {
		result = plural(int(weeks), "week") + plural(int(days), "day") + plural(int(hours), "hour") + plural(int(minutes), "minute") + plural(int(seconds), "second")
	} else if days > 0 {
		result = plural(int(days), "day") + plural(int(hours), "hour") + plural(int(minutes), "minute") + plural(int(seconds), "second")
	} else if hours > 0 {
		result = plural(int(hours), "hour") + plural(int(minutes), "minute") + plural(int(seconds), "second")
	} else if minutes > 0 {
		result = plural(int(minutes), "minute") + plural(int(seconds), "second")
	} else {
		result = plural(int(seconds), "second")
	}

	return
}

func upTimeToHuman(time int) string {
	return secondsToHuman(time / 100)
}

func Dump(obj interface{}) {
	fmt.Printf("\n%#v\n", obj)
}

func RunCommand(command string) (string, error) {
	defer func() {

		if a := recover(); a != nil {
			fmt.Println("Panic with ", a)
			log.Println("Panic: ", a)
			//main()
		}
	}()

	cmd := exec.Command("cmd", "/C", command)
	if cmd != nil {
		out, err := cmd.Output()
		if err != nil {
			return "", errors.New("Command didn't executed, error in .Output")
		} else {
			return string(out), nil
		}

	}
	return "", errors.New("Command didn't executed, error in exec.Command")
}

func WriteStrToFile(fileName, text string) bool {
	f, err := os.Create(fileName)
	if err != nil {
		fmt.Println(err)
		return false
	}
	_, err = f.WriteString(text)
	if err != nil {
		fmt.Println(err)
		f.Close()
		return false
	}

	err = f.Close()
	if err != nil {
		fmt.Println(err)
		return false
	}
	return true
}

func RandStringRunes(n int) string {
	rand.Seed(time.Now().UnixNano())
	var letterRunes = []rune("abcdefghijklmnopqrstuvwxyzABCDEFGHIJKLMNOPQRSTUVWXYZ")

	b := make([]rune, n)
	for i := range b {
		b[i] = letterRunes[rand.Intn(len(letterRunes))]
	}
	return string(b)
}
