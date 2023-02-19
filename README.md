# CloudSonic

CloudSonic is a tool that scans Cloudflare IP addresses to find an IP with low response time. The tool reads IP addresses from a file and sends HTTPS requests to each IP. The response time is measured and the results are categorized based on response time (less than 500ms, between 500ms and 1s, and greater than 1s).

## Usage

go run main.go <filename> <hostname>

- `<filename>`: Path to a file containing a list of IP addresses (one per line)
- `<hostname>`: The target hostname to be used in the HTTPS requests

## Output

The tool creates a directory with the current date and saves the following CSV files:

- `500.csv`: IPs with response time less than 500ms
- `1000.csv`: IPs with response time between 500ms and 1s
- `slowResponses.csv`: IPs with response time greater than 1s
- `errors.csv`: IPs that failed to respond
