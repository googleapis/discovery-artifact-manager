// Copyright 2020 Google LLC
//
// Licensed under the Apache License, Version 2.0 (the "License");
// you may not use this file except in compliance with the License.
// You may obtain a copy of the License at
//
//      http://www.apache.org/licenses/LICENSE-2.0
//
// Unless required by applicable law or agreed to in writing, software
// distributed under the License is distributed on an "AS IS" BASIS,
// WITHOUT WARRANTIES OR CONDITIONS OF ANY KIND, either express or implied.
// See the License for the specific language governing permissions and
// limitations under the License.

const path = require('path');
const fs = require('fs');
const {default: Q} = require('p-queue');
const {request} = require('gaxios');

const DISCOVERY_URL = 'https://www.googleapis.com/discovery/v1/apis/';
const DOWNLOAD_PATH = "discoveries";

/**
 * Download all discovery documents into the /discovery directory.
 */
async function downloadDiscoveryDocs() {
  const headers = {'X-User-Ip': '0.0.0.0'};
  console.log(`sending request to ${DISCOVERY_URL}`);
  const res = await request({url: DISCOVERY_URL, headers});
  const apis = res.data.items;
  const indexPath = path.join(DOWNLOAD_PATH, 'index.json');
  fs.writeFileSync(indexPath, JSON.stringify(res.data, null, 2));
  const queue = new Q({concurrency: 25});
  console.log(`Downloading ${apis.length} APIs...`);
  const changes = await queue.addAll(
    apis.map(api => async () => {
      console.log(`Downloading ${api.id}...`);
      const apiPath = path.join(
        DOWNLOAD_PATH,
        api.id.replace(':', '.') + '.json'
      );
      const url = api.discoveryRestUrl;
      const changeSet = {api, changes: []};
      try {
        const res = await request({url});
        // The keys in the downloaded JSON come back in an arbitrary order from
        // request to request. Sort them before storing.
        const newDoc = sortKeys(res.data);
        let updateFile = true;

        try {
          const oldDoc = JSON.parse(fs.readFileSync(apiPath, 'utf8'));
          updateFile = shouldUpdate(newDoc, oldDoc);
          changeSet.changes = getDiffs(oldDoc, newDoc);
        } catch {
          // If the file doesn't exist, that's fine it's just new
        }
        if (updateFile) {
          fs.writeFileSync(apiPath, JSON.stringify(newDoc, null, 2));
        }
      } catch (e) {
        console.error(`Error downloading: ${url}`);
      }
      return changeSet;
    })
  );
  return changes;
}

const ignoreLines = /^\s+"(?:etag|revision)": ".+"/;

/**
 * Determine if any of the changes in the discovery docs were interesting
 * @param newDoc New downloaded schema
 * @param oldDoc The existing schema from disk
 */
function shouldUpdate(newDoc, oldDoc) {
  const [newLines, oldLines] = [newDoc, oldDoc].map(doc =>
    JSON.stringify(doc, null, 2)
      .split('\n')
      .filter(l => !ignoreLines.test(l))
      .join('\n')
  );
  return newLines !== oldLines;
}

/**
 * Given an arbitrary object, recursively sort the properties on the object
 * by the name of the key.  For example:
 * {
 *   b: 1,
 *   a: 2
 * }
 * becomes....
 * {
 *   a: 2,
 *   b: 1
 * }
 * @param obj Object to be sorted
 * @returns object with sorted keys
 */
function sortKeys(obj) {
  const sorted = {};
  let keys = Object.keys(obj);
  keys = keys.sort();
  for (const key of keys) {
    // typeof [] === 'object', which is maddening
    if (!Array.isArray(obj[key]) && typeof obj[key] === 'object') {
      sorted[key] = sortKeys(obj[key]);
    } else {
      sorted[key] = obj[key];
    }
  }
  return sorted;
}

downloadDiscoveryDocs();
