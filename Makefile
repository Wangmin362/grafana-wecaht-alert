
all:
	go build -gcflags=all="-N -l" -o grafanaWebhook ./

clean:
	rm -f grafanaWebhook

copy:
	scp -r ./grafanaWebhook 172.30.3.192:/tmp
