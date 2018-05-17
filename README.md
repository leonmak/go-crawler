# Golang Web Crawler

## Extensions
- Use Goroutines & Channels for concurrency
- Output to csv

## Getting Started
- Install:
    - [go](https://golang.org/doc/install)
    - [python3](https://www.python.org/downloads/release/python-364/) to host local files
- Run:
    - Multiple Sites:
    ```
    sudo go run main.go https://golang.org https://google.com
    ```
    - Single Site:
    ```
    sudo go run main.go https://google.com
    ```
    - Custom Depth (default is 1, starts at 0):
    ```
    sudo go run main.go --depth 2 https://golang.org https://google.com
    ```
    - Tests:
    ```
    # start server
    cd test-site && python3 -m http.server &

    # run script
    cd .. && sudo go run main.go http://localhost:8000/ http://localhost:8000/2.html http://localhost:8000/1.html

    # stop server
    ps | grep 'Python -m http.server' | awk 'NR==1{print $1}' | xargs kill
    ```

### Options
`main.main()`:
-	`log.SetPriorityString("info")`
