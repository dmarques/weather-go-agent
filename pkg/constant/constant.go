package constant

import "time"

const (
	MaxMessageBytes       = 4 * 1024         // 4KB
	MaxLocationLen        = 100              // 100 bytes
	MaxToolCalls          = 2                // 2 tool calls per request
	MaxOutputTokens       = 300              // 300 tokens
	MaxConcurrentRequests = 20               // 20 concurrent requests
	RequestTimeout        = 20 * time.Second // 20 seconds
	PerIPLimit            = 20               // 20 requests per minute
	PerIPWindow           = time.Minute      // 1 minute
)
