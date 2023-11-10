package qbittorrent

type GlobalTransferInfo struct {
	DownloadSpeed  int    `json:"dl_info_speed"`
	DownloadedData int    `json:"dl_info_data"`
	UploadSpeed    int    `json:"up_info_speed"`
	UploadedData   int    `json:"up_info_data"`
	DhtNodes       int    `json:"dht_nodes"`
	Status         string `json:"connection_status"`
}
