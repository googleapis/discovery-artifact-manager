# Copyright 2022 Google LLC
#
# Licensed under the Apache License, Version 2.0 (the "License");
# you may not use this file except in compliance with the License.
# You may obtain a copy of the License at
#
#     https://www.apache.org/licenses/LICENSE-2.0
#
# Unless required by applicable law or agreed to in writing, software
# distributed under the License is distributed on an "AS IS" BASIS,
# WITHOUT WARRANTIES OR CONDITIONS OF ANY KIND, either express or implied.
# See the License for the specific language governing permissions and
# limitations under the License.

import glob
import json
import logging
import os
import os.path
import sys
import urllib.request
from typing import Any, Optional


def main() -> None:
    """The entrypoint for updating discovery documents

    This script pulls the latest discovery documents from the discovery service
    and makes any necessary changes to update the discoveries directory.
    """
    logging.basicConfig(level=logging.INFO)
    os.chdir("discoveries")
    index_document: DocumentInfo = load_index()
    service_documents: list[DocumentInfo] = load_documents(index_document)
    delete_unused_files(index_document, service_documents)
    update_files(service_documents)
    update_index(index_document)
    os.chdir("..")


class DocumentInfo:
    """An object that provides information about a discovery document

    This could be a discovery document or a discovery index. The former will
    have a revision string, whereas the latter will set it to None.

    Attributes:
        filename {str} -- The name of the json file in the discoveries dir
        content {bytes} -- The raw source
        json {object} -- The parsed JSON object, or None if parsing failed
        revision {str} -- The document revision, or None if this is an index
        json_without_revision {object} -- The parsed JSON object with revision
            and etag fields removed, for semantic comparisons
        json_string {str} -- The JSON object serialised to string, with keys sorted
    """

    filename: str
    content: bytes
    json: Any
    revision: Optional[str]
    json_without_revision: Any
    json_string: str

    def __init__(self, content: bytes, filename: Optional[str] = None) -> None:
        """Analyzes document content

        Arguments:
            content {bytes} -- The raw content from a file or HTTP response
            filename {str} -- The filename (defaults to the index filename)
        """
        self.filename = filename or "index.json"
        self.content = content
        try:
            self.json = json.loads(content)
            self.revision = self.json.get("revision")
            self.json_without_revision = self.json.copy()
            self.json_string = json.dumps(self.json, indent=2, sort_keys=True)
            if "revision" in self.json_without_revision:
                del self.json_without_revision["revision"]
            if "etag" in self.json_without_revision:
                del self.json_without_revision["etag"]
        except json.decoder.JSONDecodeError:
            self.json = None
            self.json_without_revision = None
            self.revision = None
            self.json_string = ""


def load_index() -> DocumentInfo:
    """Downloads the current discovery index and returns a document

    Returns:
        DocumentInfo -- Analysis of the discovery index content
    """
    print("LOADING index document ...")
    with urllib.request.urlopen(
        "https://discovery.googleapis.com/discovery/v1/apis"
    ) as f:
        if f.status != 200:
            sys.exit(f"Got HTTP status {f.status} for discovery index")
        document: DocumentInfo = DocumentInfo(f.read())
    if document.json is None:
        sys.exit("Unable to parse discovery index")
    logging.info("Loaded index")
    return document


def load_documents(index_document: DocumentInfo) -> list[DocumentInfo]:
    """Downloads all discovery documents in the index

    Arguments:
        index_document {DocumentInfo} -- The index

    Returns:
        list[DocumentInfo] -- List of discovery document content
    """
    print("LOADING service documents ...")
    service_documents: list[DocumentInfo] = []
    for item in index_document.json["items"]:
        name: str = item["name"]
        version: str = item["version"]
        discovery_rest_url: str = item["discoveryRestUrl"]
        filename: str = f"{name}.{version}.json"
        # Sometimes the index lists services that don't exist. So log any
        # errors but don't let them crash the entire script.
        try:
            with urllib.request.urlopen(discovery_rest_url) as f:
                if f.status != 200:
                    logging.error(
                        f"HTTP status {f.status} when loading {filename} from {discovery_rest_url}"
                    )
                    continue
                document: DocumentInfo = DocumentInfo(f.read(), filename)
        except urllib.error.HTTPError:
            logging.error(f"Failed to load {filename} from {discovery_rest_url}")
            continue
        if document.revision == None:
            logging.error(
                f"Malformed document for {filename} from {discovery_rest_url}"
            )
            continue
        logging.info(f"Loaded {filename} from {discovery_rest_url}")
        service_documents.append(document)
    return service_documents


def delete_unused_files(
    index_document: DocumentInfo, service_documents: list[DocumentInfo]
) -> None:
    """Deletes files in the discoveries directory that are no longer used

    Arguments:
        index_document {DocumentInfo} -- The index
        service_documents {Sequence[DocumentInfo]} -- The discoveries
    """
    expected_names: set[str] = set([index_document.filename])
    for doc in service_documents:
        expected_names.add(doc.filename)
    for filename in glob.glob("*.json"):
        if filename in expected_names:
            continue
        print(f"REMOVING file {filename}")
        os.remove(filename)


def update_files(service_documents: list[DocumentInfo]) -> None:
    """Iterates the service discovery documents and updates the discoveries dir

    This updates only files where we downloaded a revision strictly greater
    than the existing revision, AND whose content has semantically changed.

    Arguments:
        service_documents {list[DocumentInfo]} -- The discoveries
    """
    for document in service_documents:
        document_revision: Optional[str] = document.revision
        assert document_revision is not None
        filename: str = document.filename
        if os.path.isfile(filename):
            with open(filename, "rb+") as f:
                existing: DocumentInfo = DocumentInfo(f.read(), filename)
                existing_revision: str = existing.revision or "(unknown)"
                if existing_revision > document_revision:
                    logging.info(
                        f"Existing revision {existing_revision} > updated revision {document_revision} for {filename}"
                    )
                    continue
                elif existing_revision == document_revision:
                    logging.info(
                        f"No change to revision {document_revision} for {filename}"
                    )
                    continue
                elif existing.json_without_revision == document.json_without_revision:
                    logging.info(
                        f"Revision updated {existing_revision} to {document_revision} but no other changes for {filename}"
                    )
                    continue
                print(
                    f"UPDATING revision {existing_revision} to {document_revision} for {filename}"
                )
                f.seek(0)
                f.truncate()
                f.write(document.json_string.encode())
        else:
            print(f"WRITING new file {filename} at revision {document_revision}")
            with open(filename, "wb") as f:
                f.write(document.json_string.encode())


def update_index(index_document: DocumentInfo) -> None:
    """Updates the index file in the discoveries dir

    Arguments:
        index_document {DocumentInfo} -- The index
    """
    filename = index_document.filename
    with open(filename, "rb+") as f:
        existing_content = f.read()
        if existing_content == index_document.content:
            logging.info(f"Index file {filename} is unchanged")
        else:
            print(f"UPDATING index file {filename}")
            f.seek(0)
            f.truncate()
            f.write(index_document.json_string.encode())


if __name__ == "__main__":
    main()
