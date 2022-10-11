package main

import (
	"flag"
	"fmt"
	"math"
	"net/url"
	"os"
	"time"

	"github.com/briandowns/spinner"
	"github.com/dustin/go-humanize"
	"github.com/glentiki/hdrhistogram"
	"github.com/jbenet/goprocess"
	"github.com/olekukonko/tablewriter"
	"github.com/ttacon/chalk"
	"github.com/valyala/fasthttp"
)

type resp struct {
	status  int
	latency int64
	size    int
}

func main() {
	uri := flag.String("uri", "", "The uri to benchmark against. (Required)")
	clients := flag.Int("connections", 10, "The number of connections to open to the server.")
	pipeliningFactor := flag.Int("pipelining", 6, "The number of pipelined requests to use.")
	runtime := flag.Int("duration", 75, "The number of seconds to run the autocannnon.")
	timeout := flag.Int("timeout", 15, "The number of seconds before timing out on a request.")
	debug := flag.Bool("debug", true, "A utility debug flag.")
	perSecond := flag.Int("persecond", 1, "The number of requests per second for uri")
	flag.Parse()

	*uri = "http://localhost:8491/v10/query/Example?v=city&v=siblings_3&v=sex"

	if *uri == "" {
		flag.PrintDefaults()
		os.Exit(1)
	}

	fmt.Printf("running %vs test @ %v\n", *runtime, *uri)
	fmt.Printf("%v connections with %v pipelining factor.\n", *clients, *pipeliningFactor)

	proc := goprocess.Background()

	respChan, errChan := runClients(proc, *clients, *pipeliningFactor, time.Second*time.Duration(*timeout), *uri, *perSecond)

	latencies := hdrhistogram.New(1, 10000, 5)
	requests := hdrhistogram.New(1, 1000000, 5)
	throughput := hdrhistogram.New(1, 100000000000, 5)

	var bytes int64 = 0
	var totalBytes int64 = 0
	var respCounter int64 = 0
	var totalResp int64 = 0

	resp2xx := 0
	respN2xx := 0

	errors := 0
	timeouts := 0

	ticker := time.NewTicker(time.Second)
	runTimeout := time.NewTimer(time.Second * time.Duration(*runtime))

	spin := spinner.New(spinner.CharSets[14], 100*time.Millisecond)
	spin.Suffix = " Running Autocannon..."
	spin.Start()

	for {
		select {
		case err := <-errChan:
			errors++
			if *debug {
				fmt.Printf("there was an error: %s\n", err.Error())
			}
			if err == fasthttp.ErrTimeout {
				timeouts++
			}
		case res := <-respChan:
			s := int64(res.size)
			bytes += s
			totalBytes += s
			respCounter++

			totalResp++
			if res.status >= 200 && res.status < 300 {
				latencies.RecordValue(int64(res.latency))
				resp2xx++
			} else {
				respN2xx++
			}

		case <-ticker.C:
			requests.RecordValue(respCounter)
			respCounter = 0
			throughput.RecordValue(bytes)
			bytes = 0
			// fmt.Println("done ticking")
		case <-runTimeout.C:
			spin.Stop()

			fmt.Println("")
			fmt.Println("")
			shortLatency := tablewriter.NewWriter(os.Stdout)
			shortLatency.SetRowSeparator("-")
			shortLatency.SetHeader([]string{
				"Stat",
				"2.5%",
				"50%",
				"97.5%",
				"99%",
				"Avg",
				"Stdev",
				"Max",
			})
			shortLatency.SetHeaderColor(tablewriter.Colors{tablewriter.Bold, tablewriter.FgCyanColor},
				tablewriter.Colors{tablewriter.Bold, tablewriter.FgCyanColor},
				tablewriter.Colors{tablewriter.Bold, tablewriter.FgCyanColor},
				tablewriter.Colors{tablewriter.Bold, tablewriter.FgCyanColor},
				tablewriter.Colors{tablewriter.Bold, tablewriter.FgCyanColor},
				tablewriter.Colors{tablewriter.Bold, tablewriter.FgCyanColor},
				tablewriter.Colors{tablewriter.Bold, tablewriter.FgCyanColor},
				tablewriter.Colors{tablewriter.Bold, tablewriter.FgCyanColor})
			shortLatency.Append([]string{
				chalk.Bold.TextStyle("Latency"),
				fmt.Sprintf("%v ms", latencies.ValueAtPercentile(2.5)),
				fmt.Sprintf("%v ms", latencies.ValueAtPercentile(50)),
				fmt.Sprintf("%v ms", latencies.ValueAtPercentile(97.5)),
				fmt.Sprintf("%v ms", latencies.ValueAtPercentile(99)),
				fmt.Sprintf("%.2f ms", latencies.Mean()),
				fmt.Sprintf("%.2f ms", latencies.StdDev()),
				fmt.Sprintf("%v ms", latencies.Max()),
			})
			shortLatency.Render()
			fmt.Println("")
			fmt.Println("")

			requestsTable := tablewriter.NewWriter(os.Stdout)
			requestsTable.SetRowSeparator("-")
			requestsTable.SetHeader([]string{
				"Stat",
				"1%",
				"2.5%",
				"50%",
				"97.5%",
				"Avg",
				"Stdev",
				"Min",
			})
			requestsTable.SetHeaderColor(tablewriter.Colors{tablewriter.Bold, tablewriter.FgCyanColor},
				tablewriter.Colors{tablewriter.Bold, tablewriter.FgCyanColor},
				tablewriter.Colors{tablewriter.Bold, tablewriter.FgCyanColor},
				tablewriter.Colors{tablewriter.Bold, tablewriter.FgCyanColor},
				tablewriter.Colors{tablewriter.Bold, tablewriter.FgCyanColor},
				tablewriter.Colors{tablewriter.Bold, tablewriter.FgCyanColor},
				tablewriter.Colors{tablewriter.Bold, tablewriter.FgCyanColor},
				tablewriter.Colors{tablewriter.Bold, tablewriter.FgCyanColor})
			requestsTable.Append([]string{
				chalk.Bold.TextStyle("Req/Sec"),
				fmt.Sprintf("%v", requests.ValueAtPercentile(1)),
				fmt.Sprintf("%v", requests.ValueAtPercentile(2.5)),
				fmt.Sprintf("%v", requests.ValueAtPercentile(50)),
				fmt.Sprintf("%v", requests.ValueAtPercentile(97.5)),
				fmt.Sprintf("%.2f", requests.Mean()),
				fmt.Sprintf("%.2f", requests.StdDev()),
				fmt.Sprintf("%v", requests.Min()),
			})
			requestsTable.Append([]string{
				chalk.Bold.TextStyle("Bytes/Sec"),
				fmt.Sprintf("%v", humanize.Bytes(uint64(throughput.ValueAtPercentile(1)))),
				fmt.Sprintf("%v", humanize.Bytes(uint64(throughput.ValueAtPercentile(2.5)))),
				fmt.Sprintf("%v", humanize.Bytes(uint64(throughput.ValueAtPercentile(50)))),
				fmt.Sprintf("%v", humanize.Bytes(uint64(throughput.ValueAtPercentile(97.5)))),
				fmt.Sprintf("%v", humanize.Bytes(uint64(throughput.Mean()))),
				fmt.Sprintf("%v", humanize.Bytes(uint64(throughput.StdDev()))),
				fmt.Sprintf("%v", humanize.Bytes(uint64(throughput.Min()))),
			})
			requestsTable.Render()

			fmt.Println("")
			fmt.Println("Req/Bytes counts sampled once per second.")
			fmt.Println("")
			fmt.Println("")
			fmt.Printf("%v 2xx responses, %v non 2xx responses.\n", resp2xx, respN2xx)
			fmt.Printf("%v total requests in %v seconds, %s read.\n", formatBigNum(float64(totalResp)), *runtime, humanize.Bytes(uint64(totalBytes)))
			if errors > 0 {
				fmt.Printf("%v total errors (%v timeouts).\n", formatBigNum(float64(errors)), formatBigNum(float64(timeouts)))
			}
			fmt.Println("Done!")

			os.Exit(0)
		}
	}
}

func formatBigNum(i float64) string {
	if i < 1000 {
		return fmt.Sprintf("%.0f", i)
	}
	return fmt.Sprintf("%.0fk", math.Round(i/1000))
}

func runClients(ctx goprocess.Process, clients int, pipeliningFactor int, timeout time.Duration, uri string, perSecond int) (<-chan *resp, <-chan error) {
	respChan := make(chan *resp, 2*clients*pipeliningFactor)
	errChan := make(chan error, 2*clients*pipeliningFactor)

	// uriList := [...]string{
	// 	"http://localhost:8491/v10/query/Example?v=city&v=siblings_3&v=sex",
	// 	"http://localhost:8491/v10/query/Example?v=city&v=siblings_3",
	// 	"http://localhost:8491/v10/query/Example?v=city",
	// }
	uriList := [...]string{
		"http://cantabular-server-publishing.internal.staging:14000/v10/query/People-Households?v=AGE",
		"http://cantabular-server-publishing.internal.staging:14000/v10/query/People-Households?v=AGEHRP",
		"http://cantabular-server-publishing.internal.staging:14000/v10/query/People-Households?v=AGGDTWPEW11G",
		"http://cantabular-server-publishing.internal.staging:14000/v10/query/People-Households?v=AHCHUK11",
		"http://cantabular-server-publishing.internal.staging:14000/v10/query/People-Households?v=OA",
		"http://cantabular-server-publishing.internal.staging:14000/v10/query/People-Households?v=CARER",
		"http://cantabular-server-publishing.internal.staging:14000/v10/query/People-Households?v=CARSNOC",
		"http://cantabular-server-publishing.internal.staging:14000/v10/query/People-Households?v=CENHEAT",
		"http://cantabular-server-publishing.internal.staging:14000/v10/query/People-Households?v=COBG",
		"http://cantabular-server-publishing.internal.staging:14000/v10/query/People-Households?v=COBHUKRC",
		"http://cantabular-server-publishing.internal.staging:14000/v10/query/People-Households?v=DISABILITY",
		"http://cantabular-server-publishing.internal.staging:14000/v10/query/People-Households?v=ECOPUK",
		"http://cantabular-server-publishing.internal.staging:14000/v10/query/People-Households?v=ETHHUK11",
		"http://cantabular-server-publishing.internal.staging:14000/v10/query/People-Households?v=ETHNICITYEW",
		"http://cantabular-server-publishing.internal.staging:14000/v10/query/People-Households?v=HEALTH",
		"http://cantabular-server-publishing.internal.staging:14000/v10/query/People-Households?v=HLQPUK11",
		"http://cantabular-server-publishing.internal.staging:14000/v10/query/People-Households?v=HOURS",
		"http://cantabular-server-publishing.internal.staging:14000/v10/query/People-Households?v=HRP",
		"http://cantabular-server-publishing.internal.staging:14000/v10/query/People-Households?v=INDGPUK11",
		"http://cantabular-server-publishing.internal.staging:14000/v10/query/People-Households?v=LANGPRF",
		"http://cantabular-server-publishing.internal.staging:14000/v10/query/People-Households?v=MAINLANGG",
		"http://cantabular-server-publishing.internal.staging:14000/v10/query/People-Households?v=MARSTAT",
		"http://cantabular-server-publishing.internal.staging:14000/v10/query/People-Households?v=NATID_ALL",
		"http://cantabular-server-publishing.internal.staging:14000/v10/query/People-Households?v=NSSEC",
		"http://cantabular-server-publishing.internal.staging:14000/v10/query/People-Households?v=NSSHUK11",
		"http://cantabular-server-publishing.internal.staging:14000/v10/query/People-Households?v=SOCMIN",
		"http://cantabular-server-publishing.internal.staging:14000/v10/query/People-Households?v=POPBASESEC",
		"http://cantabular-server-publishing.internal.staging:14000/v10/query/People-Households?v=PSSPUK",
		"http://cantabular-server-publishing.internal.staging:14000/v10/query/People-Households?v=RELIGIONEW",
		"http://cantabular-server-publishing.internal.staging:14000/v10/query/People-Households?v=RESIDTYPE",
		"http://cantabular-server-publishing.internal.staging:14000/v10/query/People-Households?v=SCGHUK11",
		"http://cantabular-server-publishing.internal.staging:14000/v10/query/People-Households?v=SCGPUK11C",
		"http://cantabular-server-publishing.internal.staging:14000/v10/query/People-Households?v=SEX",
		"http://cantabular-server-publishing.internal.staging:14000/v10/query/People-Households?v=SIZHUK",
		"http://cantabular-server-publishing.internal.staging:14000/v10/query/People-Households?v=TENHUK11",
		"http://cantabular-server-publishing.internal.staging:14000/v10/query/People-Households?v=TRANSPORT",
		"http://cantabular-server-publishing.internal.staging:14000/v10/query/People-Households?v=TYPACCOM",
		"http://cantabular-server-publishing.internal.staging:14000/v10/query/People-Households?v=UNEMPHIST",
		"http://cantabular-server-publishing.internal.staging:14000/v10/query/People-Households?v=WELSHPUK112",
		"http://cantabular-server-publishing.internal.staging:14000/v10/query/People-Households?v=YRARRYEARG",
		"http://cantabular-server-publishing.internal.staging:14000/v10/query/Usual-Residents?v=AGE",
		"http://cantabular-server-publishing.internal.staging:14000/v10/query/Usual-Residents?v=HRP",
		"http://cantabular-server-publishing.internal.staging:14000/v10/query/Usual-Residents?v=LANGPRF",
		"http://cantabular-server-publishing.internal.staging:14000/v10/query/Usual-Residents?v=MARSTAT",
		"http://cantabular-server-publishing.internal.staging:14000/v10/query/Usual-Residents?v=NATID_ALL",
		"http://cantabular-server-publishing.internal.staging:14000/v10/query/Usual-Residents?v=NSSEC",
		"http://cantabular-server-publishing.internal.staging:14000/v10/query/Usual-Residents?v=RELIGIONEW",
		"http://cantabular-server-publishing.internal.staging:14000/v10/query/Usual-Residents?v=SEX",
		"http://cantabular-server-publishing.internal.staging:14000/v10/query/Household-Ref-Persons?v=AGE",
		"http://cantabular-server-publishing.internal.staging:14000/v10/query/Household-Ref-Persons?v=MAINLANGG",
		"http://cantabular-server-publishing.internal.staging:14000/v10/query/Household-Ref-Persons?v=TRANSPORT",
		"http://cantabular-server-publishing.internal.staging:14000/v10/query/Household-Ref-Persons?v=WELSHPUK112",
	}

	//	u, _ := url.Parse(uri)

	for i := 0; i < len(uriList); i++ {
		u, _ := url.Parse(uriList[i])
		c := fasthttp.PipelineClient{
			Addr:               getAddr(u),
			IsTLS:              u.Scheme == "https",
			MaxPendingRequests: pipeliningFactor,
			MaxConns:           1000,
		}

		for j := 0; j < pipeliningFactor; j++ {
			go func(i int) {
				req := fasthttp.AcquireRequest()
				req.SetBody([]byte("hello, world!"))
				req.SetRequestURI(uriList[i])
				res := fasthttp.AcquireResponse()

				for {
					// we have ~ 51 different uri's to make requests of, so a request period of just over 5 seconds
					// will limit the rate to about 10 per second MULTIPLIED by the 'pipeliningFactor' number ...
					// OR multiplied by how many hosts this is run from .
					perSecondEndTime := time.Now().UnixMilli() + 5050

					// each uri is request at an offset time to spread the load out
					time.Sleep(time.Duration(time.Duration(i) * 40 * time.Millisecond))

					for j := 0; j < perSecond; j++ {
						startTime := time.Now()
						if err := c.DoTimeout(req, res, timeout); err != nil {
							fmt.Printf("\nerror for uri: %s, %v\n", uriList[i], err)
							errChan <- err
						} else {
							size := len(res.Body()) + 2
							res.Header.VisitAll(func(key, value []byte) {
								size += len(key) + len(value) + 2
							})

							s := res.Body()
							l := len(s)
							if l > 20 {
								l = 20
							}
							// print first 20 chars ...
							fmt.Printf("body: %s\n", s[:l])
							respChan <- &resp{
								status:  res.Header.StatusCode(),
								latency: time.Since(startTime).Milliseconds(),
								size:    size,
							}
							res.Reset()
						}
					}
					left := time.Now().UnixMilli()
					if left < perSecondEndTime {
						sleepTime := perSecondEndTime - left
						fmt.Printf("Doing sleep (%d): %d\n", i, sleepTime)
						time.Sleep(time.Duration(sleepTime) * time.Millisecond)
					}
				}
			}(i)
		}
	}
	return respChan, errChan
}

// getAddr returns the address from a URL, including the port if it's not empty.
// So it can return hostname:port or simply hostname
func getAddr(u *url.URL) string {
	if u.Port() == "" {
		return u.Hostname()
	} else {
		return fmt.Sprintf("%v:%v", u.Hostname(), u.Port())
	}
}
