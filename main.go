package main

import (
	"bufio"
	"context"
	"crypto/tls"
	"encoding/csv"
	"fmt"
	"net"
	"net/http"
	"os"
	"strconv"
	"strings"
	"sync"
	"time"
)

func main() {
	args := os.Args[1:]
	if len(args) < 2 {
		fmt.Println("Usage: go run main.go <filename> <hostname>")
	}
	filename := args[0]
	hostname := args[1]
	MaxOpenConnections := 5000
	BatchSize := 200
	writeSize := 10

	ips := readIps(filename)
	var wg sync.WaitGroup
	wg.Add(len(ips))
	ctx, cancel := context.WithCancel(context.Background())
	defer cancel()
	sem := make(chan struct{}, MaxOpenConnections)
	sem2 := make(chan struct{}, BatchSize)
	results := make(chan string, 1000)
	errors := make(chan string, 1000)
	errorBatch := make([][]string, 0, 1000)
	go func() {

		for e := range errors {
			errorBatch = append(errorBatch, []string{e})
			if len(errorBatch) == writeSize {
				writeResults("errors.csv", errorBatch)
				errorBatch = make([][]string, 0, 1000)
			}
		}
	}()
	go func() {
		resultsBatch500 := make([][]string, 0, BatchSize)
		resultsBatch1000 := make([][]string, 0, BatchSize)
		resultsBatchWasted := make([][]string, 0, BatchSize)

		for result := range results {
			rSlice := strings.Split(strings.ReplaceAll(result, "\"", ""), ",")
			respTime, err := strconv.Atoi(rSlice[1])
			if err != nil {
				fmt.Println(err)
			}

			if respTime < 501 {
				resultsBatch500 = append(resultsBatch500, rSlice)
			} else if respTime < 1001 {
				resultsBatch1000 = append(resultsBatch1000, rSlice)
			} else {
				resultsBatchWasted = append(resultsBatchWasted, rSlice)
			}

			if len(resultsBatch500) >= writeSize {
				writeResults("500.csv", resultsBatch500)
				resultsBatch500 = make([][]string, 0, BatchSize)
			}
			if len(resultsBatch1000) >= writeSize {
				writeResults("1000.csv", resultsBatch1000)
				resultsBatch1000 = make([][]string, 0, BatchSize)
			}
			if len(resultsBatchWasted) >= writeSize {
				writeResults("others.csv", resultsBatchWasted)
				resultsBatchWasted = make([][]string, 0, BatchSize)
			}
		}
	}()
	for _, ip := range ips {
		sem <- struct{}{}
		sem2 <- struct{}{}
		go func(ip string) {
			defer func() {
				<-sem
				<-sem2
				wg.Done()
			}()
			result, err := request(ctx, ip, hostname)
			if err != nil {
				errors <- err.Error()
			} else {
				results <- result
			}

		}(ip)
	}
	wg.Wait()
	close(results)
}

func readIps(filename string) []string {
	file, err := os.Open(filename)
	if err != nil {
		panic(err)
	}
	defer file.Close()

	var ips []string
	scanner := bufio.NewScanner(file)
	for scanner.Scan() {
		ips = append(ips, scanner.Text())
	}
	return ips
}

func request(ctx context.Context, ip string, hostname string) (string, error) {

	conn := &tls.Config{
		ServerName: hostname,
	}
	start := time.Now()
	tlsConn, err := tls.DialWithDialer(&net.Dialer{Timeout: 1 * time.Second}, "tcp", ip+":443", conn)

	if err != nil {
		return fmt.Sprintf("error: %s, %v", ip, err), err
	}
	tlsConn.Write([]byte("GET / HTTP/1.1\r Host: " + hostname + "\r"))
	_, err = http.ReadResponse(bufio.NewReader(tlsConn), nil)
	if err != nil {
		return fmt.Sprintf("error: %s, %v", ip, err), err
	}

	defer tlsConn.Close()
	elapsed := time.Since(start)

	return fmt.Sprintf("%s,%v", ip, elapsed.Milliseconds()), err

}

func writeResults(filename string, results [][]string) {
	t := time.Now()
	dir := fmt.Sprintf("%d-%02d-%02d", t.Year(), t.Month(), t.Day())
	if _, err := os.Stat(dir); os.IsNotExist(err) {
		os.Mkdir(dir, 0755)
	}
	filename = dir + "/" + filename
	file, err := os.OpenFile(filename, os.O_APPEND|os.O_CREATE|os.O_WRONLY, 0644)
	if err != nil {
		panic(err)
	}
	defer file.Close()
	writer := csv.NewWriter(file)
	defer writer.Flush()
	for _, result := range results {
		err := writer.Write(result)
		if err != nil {
			panic(err)
		}
	}
}
