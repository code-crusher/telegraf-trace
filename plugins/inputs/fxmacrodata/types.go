package fxmacrodata

// series is one of the endpoints requested in every gather cycle. The list is
// built once in Init from the configured currencies, indicators and currency
// pairs.
type series struct {
	endpoint string
	name     string
	field    string
	tags     map[string]string
}

// seriesResponse is the common shape of the endpoints queried by this plugin.
// The announcements and forex endpoints share the relevant fields: the data
// rows with the value and publication instant, plus the publishing source.
type seriesResponse struct {
	Source string      `json:"source"`
	Data   []dataPoint `json:"data"`
}

// dataPoint is a single observation of a series. The value and the
// publication instant are both nullable in the API response, as the
// publishing authority may not report a value for a period and may not state
// when a figure was published.
type dataPoint struct {
	Date                 string   `json:"date"`
	Value                *float64 `json:"val"`
	AnnouncementDatetime *int64   `json:"announcement_datetime"`
}
