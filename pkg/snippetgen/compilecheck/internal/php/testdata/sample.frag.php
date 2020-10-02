
<?php
/*
 * PRE-REQUISITES:
 * ---------------
 * 1. If not already done, enable the Google App Engine Admin API and check quota for your project at
 *    https://console.developers.google.com/apis/api/appengine_component/quotas
 * 2. To install the client library on Composer, check installation instructions at
 *    https://github.com/google/google-api-php-client.
 * 3. This sample uses Application Default Credentials for Auth. If not already done, install the gcloud CLI from
 *    https://cloud.google.com/sdk/ and run 'gcloud beta auth application-default login'
 */

// composer autoloading
require_once __DIR__ . '/vendor/autoload.php';


// Create a new client
$client = new Google_Client();
$client->setApplicationName('Client Sample Application');
$client->useApplicationDefaultCredentials();
$client->addScope('https://www.googleapis.com/auth/cloud-platform');

// Create a new Appengine service
$service = new Google_Service_Appengine($client);

// Part of `name`. Name of the application to get. For example: "apps/myapp".
$appsId = '';

$myFoo = 0;

$requestBody = object();
$requestBody->setMyFoo($myFoo);

// TODO: To download media content, use:
//
// $optParams['alt'] = 'media';

do {
  $response = $service->apps->get($appsId, $requestBody);
} while (true);
?>
