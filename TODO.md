# TODO
- [ ] HTTPS
- [ ] Dynamic Timeout per check?
- [x] Checks are not a central file, individual kv per check
- [x] Results have pass/fail in stream name


## Agent
- [x] Dynamic Team resources (ip, team#, etc.)
  - [x] User input can contain templated values. This should be blocked (the janky way)
- [x] Overhaul input instead of env vars (nats, team#)
- [ ] Cleanup main cmd function

## Frontend
- [ ] Admin dashboard
- [ ] Admin settings
  - [x] Start/Stop scoring buttons
  - [x] View global settings
- [ ] Observer dashboard
- [ ] Grouping NATS data points. Must be a better way
- [x] Team ID from JWT

## Server
- [ ] Additional Admin routes
  - Update settings?
- [x] Get team ID during auth
- [ ] Flip static_auth to be token:role (would allow multiple of the same account with different tokens)




```go
	log.SetOutput(os.Stdout)
	log.SetFlags(log.LstdFlags | log.Lshortfile)

	// Load configuration
	cfg, err := config.Load("config.json")
	if err != nil {
		log.Fatalf("Failed to load config: %v", err)
	}

	// Connect to NATS settings KV
	var natsKVClient *nats.NATSKVClient
	natsKVClient, err = nats.NewNATSKVClient(cfg.NATSUrl, cfg.NATSCredsFile)
	if err != nil {
		log.Printf("Warning: Failed to initialize NATS KV client: %v", err)
	} else {
		log.Println("Connected to NATS KV bucket 'settings'")
		defer natsKVClient.Close()
	}

	// Setup NATS key value handler
	kv := natsKVClient.GetKVClient()

	watcher, err := kv.Watch("check.*") //kv.WatchAll()
	if err != nil {
		log.Fatal(err)
	}
	defer watcher.Stop()

	for entry := range watcher.Updates() {
		if entry == nil {
			// nil signals the end of the initial values (historical state)
			fmt.Println("--- initial state delivered ---")
			continue
		}
		fmt.Printf("op=%s key=%s value=%s revision=%d\n",
			entry.Operation(), entry.Key(), string(entry.Value()), entry.Revision())
	}
```
