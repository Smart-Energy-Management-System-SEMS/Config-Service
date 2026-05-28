# Go usage

```go
boot := configclient.LoadBootstrap()
runtimeCfg, err := configclient.FetchRuntimeConfig(context.Background(), boot)
if err != nil {
    log.Fatalf("config bootstrap failed: %v", err)
}
log.Printf("loaded config for %s", runtimeCfg.Service)
```
