package youtube

type YTMusic_Runs []struct {
	Text               string `json:"text"`
	NavigationEndpoint struct {
		WatchEndpoint struct {
			VideoId                            string `json:"videoId"`
			WatchEndpointMusicSupportedConfigs struct {
				WatchEndpointMusicConfig struct {
					MusicVideoType string `json:"musicVideoType"`
				} `json:"watchEndpointMusicConfig"`
			} `json:"watchEndpointMusicSupportedConfigs"`
		} `json:"watchEndpoint"`

		BrowseEndpoint struct {
			BrowseId                              string `json:"browseId"`
			BrowseEndpointContextSupportedConfigs struct {
				BrowseEndpointContextMusicConfig struct {
					PageType string `json:"pageType"`
				} `json:"browseEndpointContextMusicConfig"`
			} `json:"browseEndpointContextSupportedConfigs"`
		} `json:"browseEndpoint"`
	} `json:"navigationEndpoint"`
}

type YTMusic_MusicShelfContent struct {
	MusicResponsiveListItemRenderer struct {
		FlexColumns []struct {
			MusicResponsiveListItemFlexColumnRenderer struct {
				Text struct {
					Runs YTMusic_Runs `json:"runs"`
				}
			} `json:"musicResponsiveListItemFlexColumnRenderer"`
		} `json:"flexColumns"`
		PlaylistItemData struct {
			VideoId string `json:"videoId"`
		} `json:"playlistItemData"`
	} `json:"musicResponsiveListItemRenderer"`
}

type YTMusic_SearchResults struct {
	Contents struct {
		TabbedSearchResultsRenderer struct {
			Tabs []struct {
				TabRenderer struct {
					Title    string `json:"title"`
					Selected bool   `json:"selected"`
					Content  struct {
						SectionListRenderer struct {
							Contents []struct {
								MusicShelfRenderer struct {
									Title struct {
										Runs []struct {
											Text string `json:"text"`
										} `json:"runs"`
									} `json:"title"`
									Contents []YTMusic_MusicShelfContent `json:"contents"`
								} `json:"musicShelfRenderer"`
							} `json:"contents"`
						} `json:"sectionListRenderer"`
					} `json:"content"`
				} `json:"tabRenderer"`
			} `json:"tabs"`
		} `json:"tabbedSearchResultsRenderer"`
	} `json:"contents"`
}
