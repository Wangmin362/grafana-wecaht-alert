
all:
	go build -gcflags=all="-N -l" -o grafanaWebhook ./

clean: 
	rm -f grafanaWebhook