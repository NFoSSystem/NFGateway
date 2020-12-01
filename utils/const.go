package utils

import (
	"time"
)

const (
	LOCAL_INTERFACE     = "127.0.0.1"
	LOGGER_PREFIX       = "[Gateway]"
	LOGGER_PUB_SUB_CHAN = "logChan"
	REDIS_HOSTNAME      = LOCAL_INTERFACE
	REDIS_PORT          = 6379

	// DHCP Settings
	DHCP_START_IP                       = "192.168.1.115"
	DHCP_MAX_LEASE_RANGE                = 50
	DHCP_MAX_TRAMSACTION_RETRY_ATTEMPTS = 5
	DHCP_LEASE_TIME                     = 10 * time.Minute
	DHCP_CLEAN_UP_SCHEDULE              = 2 * time.Minute
)
