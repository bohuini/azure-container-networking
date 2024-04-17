package v1alpha1

// IsReady checks if all the required fields in the MTPNC status are populated
func (m *MultitenantPodNetworkConfig) IsReady() bool {
	return m.Status.PrimaryIP != "" && m.Status.MacAddress != "" && m.Status.NCID != "" && m.Status.GatewayIP != "" && m.Status.NICType != ""
}
