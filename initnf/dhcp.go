package initnf

import (
	"faasrouter/utils"
	"net"

	"bitbucket.org/Manaphy91/faasdhcp/dhcpdb"
)

func InitDhcp() {
	client := dhcpdb.NewRedisClient(utils.REDIS_HOSTNAME, 6379)

	startIp := net.ParseIP(utils.DHCP_START_IP)
	sc := dhcpdb.NewSharedContext(client, utils.DHCP_MAX_LEASE_RANGE, &startIp,
		utils.DHCP_MAX_TRAMSACTION_RETRY_ATTEMPTS)

	if err := dhcpdb.CleanUpAvailableIpRange(client); err != nil {
		utils.RLogger.Fatalln(err)
	}

	if err := dhcpdb.CleanUpIpMacMapping(client); err != nil {
		utils.RLogger.Fatalln(err)
	}

	if err := dhcpdb.CleanUpIpSets(client); err != nil {
		utils.RLogger.Fatalln(err)
	}

	if err := dhcpdb.InitAvailableIpRange(client, utils.DHCP_MAX_LEASE_RANGE); err != nil {
		utils.RLogger.Fatalln(err)
	}

	go sc.CleanUpExpiredMappings(utils.DHCP_LEASE_TIME, utils.DHCP_CLEAN_UP_SCHEDULE,
		utils.RLogger)
}
