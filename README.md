# Setup

```
  # start containers
- docker-compose up

  # to stop the container and print stats!
- docker-compose stop backend

```

## Endpoints

```go
// localhost:3000 by default
router.Handle("/health", health()) // Health check
router.Handle("/api/standing/top", top(svc)) // Top 10 words
router.Handle("/api/standing/{word}", standing(svc)) // standing +5 -5 of the {word}
```

## Misc

The avg event per minute is printed to stdout as polled
