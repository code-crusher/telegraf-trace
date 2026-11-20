//go:build !custom || serializers || serializers.elasticcommonschema

package all

import (
	_ "github.com/influxdata/telegraf/plugins/serializers/elasticcommonschema" // register plugin
)
