package test

import (
	"testing"
	"path/filepath"
	"github.com/gruntwork-io/terratest"
	terralog "github.com/gruntwork-io/terratest/log"
	"fmt"
	"github.com/gruntwork-io/terratest/test-structure"
)

const couchbaseClusterVarName = "cluster_name"

func TestIntegrationCouchbaseCommunitySingleClusterUbuntu(t *testing.T) {
	t.Parallel()
	testCouchbaseSingleCluster(t, "TestIntegrationCouchbaseCommunitySingleClusterUbuntu", "ubuntu", "community", "http")
}

func TestIntegrationCouchbaseCommunitySingleClusterAmazonLinux(t *testing.T) {
	t.Parallel()
	testCouchbaseSingleCluster(t, "TestIntegrationCouchbaseCommunitySingleClusterAmazonLinux", "amazon-linux", "community", "http")
}

func TestIntegrationCouchbaseEnterpriseSingleClusterUbuntu(t *testing.T) {
	t.Parallel()
	testCouchbaseSingleCluster(t, "TestIntegrationCouchbaseEnterpriseSingleClusterUbuntu", "ubuntu", "enterprise", "http")
}

func TestIntegrationCouchbaseEnterpriseSingleClusterAmazonLinux(t *testing.T) {
	t.Parallel()
	testCouchbaseSingleCluster(t, "TestIntegrationCouchbaseEnterpriseSingleClusterAmazonLinux", "amazon-linux", "enterprise", "http")
}

func testCouchbaseSingleCluster(t *testing.T, testName string, osName string, edition string, loadBalancerProtocol string) {
	logger := terralog.NewLogger(testName)

	examplesFolder := test_structure.CopyTerraformFolderToTemp(t, "../", "examples", testName, logger)
	couchbaseAmiDir := filepath.Join(examplesFolder, "couchbase-ami")
	couchbaseSingleClusterDir := filepath.Join(examplesFolder, "couchbase-single-cluster")

	test_structure.RunTestStage("setup_ami", logger, func() {
		testStageBuildCouchbaseAmi(t, osName, edition, couchbaseAmiDir, couchbaseSingleClusterDir, logger)
	})

	test_structure.RunTestStage("setup_deploy", logger, func() {
		resourceCollection := test_structure.LoadRandomResourceCollection(t, couchbaseSingleClusterDir, logger)
		amiId := test_structure.LoadAmiId(t, couchbaseSingleClusterDir, logger)

		terratestOptions := createBaseTerratestOptions(t, testName, couchbaseSingleClusterDir, resourceCollection)
		terratestOptions.Vars = map[string]interface{} {
			"aws_region":            resourceCollection.AwsRegion,
			"ami_id":                amiId,
			couchbaseClusterVarName: formatCouchbaseClusterName("single-cluster", resourceCollection),
		}

		deploy(t, terratestOptions)

		test_structure.SaveTerratestOptions(t, couchbaseSingleClusterDir, terratestOptions, logger)
	})

	defer test_structure.RunTestStage("teardown", logger, func() {
		testStageTeardown(t, couchbaseSingleClusterDir, logger)
	})

	defer test_structure.RunTestStage("logs", logger, func() {
		resourceCollection := test_structure.LoadRandomResourceCollection(t, couchbaseSingleClusterDir, logger)
		testStageLogs(t, couchbaseSingleClusterDir, couchbaseClusterVarName, resourceCollection, logger)
	})

	test_structure.RunTestStage("validation", logger, func() {
		terratestOptions := test_structure.LoadTerratestOptions(t, couchbaseSingleClusterDir, logger)
		clusterName := getClusterName(t, couchbaseClusterVarName, terratestOptions)

		couchbaseServerUrl, err := terratest.OutputRequired(terratestOptions, "couchbase_web_console_url")
		if err != nil {
			t.Fatal(err)
		}
		couchbaseServerUrl = fmt.Sprintf("%s://%s:%s@%s", loadBalancerProtocol, usernameForTest, passwordForTest, couchbaseServerUrl)

		syncGatewayUrl, err := terratest.OutputRequired(terratestOptions, "sync_gateway_url")
		if err != nil {
			t.Fatal(err)
		}
		syncGatewayUrl = fmt.Sprintf("%s://%s/%s", loadBalancerProtocol, syncGatewayUrl, clusterName)

		checkCouchbaseConsoleIsRunning(t, couchbaseServerUrl, logger)
		checkCouchbaseClusterIsInitialized(t, couchbaseServerUrl, 3, logger)
		checkCouchbaseDataNodesWorking(t, couchbaseServerUrl, logger)
		checkSyncGatewayWorking(t, syncGatewayUrl, logger)
	})
}
