package gcs

import (
	"bytes"
	"reflect"
	"testing"
)

func TestScanGCSTree(t *testing.T) {
	input := `
/some/path/snippet-staging/public/bigquery/:
/some/path/snippet-staging/public/bigquery/v2/

/some/path/snippet-staging/public/cloudbilling/:
/some/path/snippet-staging/public/cloudbilling/v1/

/some/path/snippet-staging/public/clouduseraccounts/:
/some/path/snippet-staging/public/clouduseraccounts/beta/

/some/path/snippet-staging/public/compute/:
/some/path/snippet-staging/public/compute/v1/

/some/path/snippet-staging/public/dataproc/:
/some/path/snippet-staging/public/dataproc/v1/
/some/path/snippet-staging/public/dataproc/v1alpha1/

/some/path/snippet-staging/public/deploymentmanager/:
/some/path/snippet-staging/public/deploymentmanager/v2/

/some/path/snippet-staging/public/dns/:
/some/path/snippet-staging/public/dns/v1/

/some/path/snippet-staging/public/logging/:
/some/path/snippet-staging/public/logging/v2beta1/

/some/path/snippet-staging/public/monitoring/:
/some/path/snippet-staging/public/monitoring/v3/

/some/path/snippet-staging/public/prediction/:
/some/path/snippet-staging/public/prediction/v1.6/

/some/path/snippet-staging/public/pubsub/:
/some/path/snippet-staging/public/pubsub/v1/

/some/path/snippet-staging/public/sqladmin/:
/some/path/snippet-staging/public/sqladmin/v1beta4/

/some/path/snippet-staging/public/storage/:
/some/path/snippet-staging/public/storage/v1/

/some/path/snippet-staging/public/storagetransfer/:
/some/path/snippet-staging/public/storagetransfer/v1/
/otherpatrh/apifoo/:

/some/path/snippet-staging/public/prediction/
/some/path/snippet-staging/public/pubsub/
/some/path/snippet-staging/public/sqladmin/
/some/path/snippet-staging/public/storage/
/some/path/snippet-staging/public/storagetransfer/
/some/path/snippet/public/dns/v1/0/
/some/path/snippet/public/dns/v1/20160224/
/some/path/snippet/public/dns/v1/20160413/
/some/path/snippet/public/dns/v1/20160601/
`
	want := []string{
		"/some/path/snippet-staging/public/bigquery/v2",
		"/some/path/snippet-staging/public/cloudbilling/v1",
		"/some/path/snippet-staging/public/clouduseraccounts/beta",
		"/some/path/snippet-staging/public/compute/v1",
		"/some/path/snippet-staging/public/dataproc/v1",
		"/some/path/snippet-staging/public/dataproc/v1alpha1",
		"/some/path/snippet-staging/public/deploymentmanager/v2",
		"/some/path/snippet-staging/public/dns/v1",
		"/some/path/snippet-staging/public/logging/v2beta1",
		"/some/path/snippet-staging/public/monitoring/v3",
		"/some/path/snippet-staging/public/prediction/v1.6",
		"/some/path/snippet-staging/public/pubsub/v1",
		"/some/path/snippet-staging/public/sqladmin/v1beta4",
		"/some/path/snippet-staging/public/storage/v1",
		"/some/path/snippet-staging/public/storagetransfer/v1",
		"/some/path/snippet-staging/public/prediction",
		"/some/path/snippet-staging/public/pubsub",
		"/some/path/snippet-staging/public/sqladmin",
		"/some/path/snippet-staging/public/storage",
		"/some/path/snippet-staging/public/storagetransfer",
		"/some/path/snippet/public/dns/v1/0",
		"/some/path/snippet/public/dns/v1/20160224",
		"/some/path/snippet/public/dns/v1/20160413",
		"/some/path/snippet/public/dns/v1/20160601",
	}

	got, err := scanGCSTree(bytes.NewBufferString(input))
	if err != nil {
		t.Errorf("unexpected error: %s", err)
	}
	if !reflect.DeepEqual(got, want) {
		t.Errorf("tree output not expected. got:\n%v", got)
	}
}
