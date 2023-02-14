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

import os
from pathlib import Path
import shutil
import unittest
from unittest.mock import MagicMock, patch

from scripts import update_disco


DISCOVERY_0001_CONTENT = b"""{
  "revision": "0001",
  "data": "foo"
}"""

DISCOVERY_0001_CONTENT_SORTED = b"""{
  "data": "foo",
  "revision": "0001"
}"""

DISCOVERY_0001A_CONTENT = b"""{
  "revision": "0001",
  "data": "bar"
}"""

DISCOVERY_0002_CONTENT = b"""{
  "revision": "0002",
  "data": "bar"
}"""

DISCOVERY_0002_CONTENT_SORTED = b"""{
  "data": "bar",
  "revision": "0002"
}"""

DISCOVERY_0003_CONTENT = b"""{
  "revision": "0003",
  "data": "bar"
}"""

INDEX_1_CONTENT = b"""{
  "items": [
    {
      "name": "service1",
      "version": "v1",
      "discoveryRestUrl": "https://example.com/service1_v1.json"
    },
    {
      "name": "service2",
      "version": "v1",
      "discoveryRestUrl": "https://example.com/service2_v1.json"
    }
  ]
}"""

INDEX_2_CONTENT = b"""{
  "items": [
    {
      "name": "service1",
      "version": "v1",
      "discoveryRestUrl": "https://example.com/service1_v1.json"
    },
    {
      "name": "service1",
      "version": "v2",
      "discoveryRestUrl": "https://example.com/service1_v2.json"
    }
  ]
}"""

INDEX_2_CONTENT_SORTED = b"""{
  "items": [
    {
      "discoveryRestUrl": "https://example.com/service1_v1.json",
      "name": "service1",
      "version": "v1"
    },
    {
      "discoveryRestUrl": "https://example.com/service1_v2.json",
      "name": "service1",
      "version": "v2"
    }
  ]
}"""

class TestUpdateDisco(unittest.TestCase):
    def setUp(self):
        self._old_path = Path.cwd()
        self._tmp_path = Path(__file__).parent / "tmp"
        if self._tmp_path.exists():
            shutil.rmtree(self._tmp_path)
        self._tmp_path.mkdir(parents=True, exist_ok=True)
        os.chdir(self._tmp_path)

    def tearDown(self):
        os.chdir(self._old_path)

    def test_document_info_interprets_discovery(self):
        doc = update_disco.DocumentInfo(DISCOVERY_0001_CONTENT, "disc.json")
        self.assertEqual("disc.json", doc.filename)
        self.assertEqual(DISCOVERY_0001_CONTENT, doc.content)
        self.assertEqual({"revision": "0001", "data": "foo"}, doc.json)
        self.assertEqual("0001", doc.revision)
        self.assertEqual({"data": "foo"}, doc.json_without_revision)

    def test_document_info_interprets_index(self):
        doc = update_disco.DocumentInfo(INDEX_1_CONTENT)
        self.assertEqual("index.json", doc.filename)
        self.assertEqual(INDEX_1_CONTENT, doc.content)
        expected_json = {
            "items": [
                {
                    "name": "service1",
                    "version": "v1",
                    "discoveryRestUrl": "https://example.com/service1_v1.json",
                },
                {
                    "name": "service2",
                    "version": "v1",
                    "discoveryRestUrl": "https://example.com/service2_v1.json",
                },
            ]
        }
        self.assertEqual(expected_json, doc.json)
        self.assertIsNone(doc.revision)
        self.assertEqual(expected_json, doc.json_without_revision)

    def test_document_info_fails_parsing(self):
        doc = update_disco.DocumentInfo("{", "disc.json")
        self.assertEqual("disc.json", doc.filename)
        self.assertEqual("{", doc.content)
        self.assertIsNone(doc.json)
        self.assertIsNone(doc.revision)
        self.assertIsNone(doc.json_without_revision)
        self.assertEqual(doc.json_string, "")

    def test_update_index(self):
        index_path = Path("index.json")
        index_path.write_bytes(INDEX_1_CONTENT)
        doc = update_disco.DocumentInfo(INDEX_2_CONTENT)
        update_disco.update_index(doc)
        self.assertEqual(INDEX_2_CONTENT_SORTED, index_path.read_bytes())

    def test_delete_unused_files(self):
        index_path = Path("index.json")
        index_path.write_bytes(INDEX_1_CONTENT)
        disc1_path = Path("disc1.json")
        disc1_path.write_bytes(DISCOVERY_0001_CONTENT)
        disc2_path = Path("disc2.json")
        disc2_path.write_bytes(DISCOVERY_0002_CONTENT)
        self.assertTrue(index_path.exists())
        self.assertTrue(disc1_path.exists())
        self.assertTrue(disc2_path.exists())
        index_doc = update_disco.DocumentInfo(INDEX_1_CONTENT)
        disc2_doc = update_disco.DocumentInfo(DISCOVERY_0002_CONTENT, "disc2.json")
        update_disco.delete_unused_files(index_doc, [disc2_doc])
        self.assertTrue(index_path.exists())
        self.assertFalse(disc1_path.exists())
        self.assertTrue(disc2_path.exists())

    def test_update_files_same_revision(self):
        disc_path = Path("disc.json")
        disc_path.write_bytes(DISCOVERY_0001_CONTENT)
        self.assertEqual(DISCOVERY_0001_CONTENT, disc_path.read_bytes())
        disc_doc = update_disco.DocumentInfo(DISCOVERY_0001A_CONTENT, "disc.json")
        update_disco.update_files([disc_doc])
        self.assertEqual(DISCOVERY_0001_CONTENT, disc_path.read_bytes())

    def test_update_files_older_revision(self):
        disc_path = Path("disc.json")
        disc_path.write_bytes(DISCOVERY_0002_CONTENT)
        self.assertEqual(DISCOVERY_0002_CONTENT, disc_path.read_bytes())
        disc_doc = update_disco.DocumentInfo(DISCOVERY_0001_CONTENT, "disc.json")
        update_disco.update_files([disc_doc])
        self.assertEqual(DISCOVERY_0002_CONTENT, disc_path.read_bytes())

    def test_update_files_newer_revision_same_data(self):
        disc_path = Path("disc.json")
        disc_path.write_bytes(DISCOVERY_0002_CONTENT)
        self.assertEqual(DISCOVERY_0002_CONTENT, disc_path.read_bytes())
        disc_doc = update_disco.DocumentInfo(DISCOVERY_0003_CONTENT, "disc.json")
        update_disco.update_files([disc_doc])
        self.assertEqual(DISCOVERY_0002_CONTENT, disc_path.read_bytes())

    def test_update_files_newer_revision_updated_data(self):
        disc_path = Path("disc.json")
        disc_path.write_bytes(DISCOVERY_0001_CONTENT)
        self.assertEqual(DISCOVERY_0001_CONTENT, disc_path.read_bytes())
        disc_doc = update_disco.DocumentInfo(DISCOVERY_0002_CONTENT, "disc.json")
        update_disco.update_files([disc_doc])
        self.assertEqual(DISCOVERY_0002_CONTENT_SORTED, disc_path.read_bytes())

    def test_update_files_new_file(self):
        disc_path = Path("disc.json")
        self.assertFalse(disc_path.exists())
        disc_doc = update_disco.DocumentInfo(DISCOVERY_0001_CONTENT, "disc.json")
        update_disco.update_files([disc_doc])
        self.assertEqual(DISCOVERY_0001_CONTENT_SORTED, disc_path.read_bytes())

    @patch("urllib.request.urlopen")
    def test_load_index(self, mock_urlopen):
        mock_urlopen.return_value.__enter__.return_value.status = 200
        mock_urlopen.return_value.__enter__.return_value.read.return_value = (
            INDEX_1_CONTENT
        )
        doc = update_disco.load_index()
        self.assertEqual(INDEX_1_CONTENT, doc.content)

    @patch("urllib.request.urlopen")
    def test_load_documents(self, mock_urlopen):
        def mock_urlopen_impl(url):
            mock_urlopen_return_value = MagicMock()
            mock_urlopen_enter_value = mock_urlopen_return_value.__enter__.return_value
            if url == "https://example.com/service1_v1.json":
                mock_urlopen_enter_value.status = 200
                mock_urlopen_enter_value.read.return_value = DISCOVERY_0001_CONTENT
            elif url == "https://example.com/service2_v1.json":
                mock_urlopen_enter_value.status = 200
                mock_urlopen_enter_value.read.return_value = DISCOVERY_0002_CONTENT
            else:
                mock_urlopen_enter_value.status = 404
                mock_urlopen_enter_value.read.return_value = ""
            return mock_urlopen_return_value

        mock_urlopen.side_effect = mock_urlopen_impl
        index_doc = update_disco.DocumentInfo(INDEX_1_CONTENT)
        docs = update_disco.load_documents(index_doc)
        self.assertEqual(2, len(docs))
        self.assertEqual(DISCOVERY_0001_CONTENT, docs[0].content)
        self.assertEqual(DISCOVERY_0002_CONTENT, docs[1].content)

    @patch("urllib.request.urlopen")
    def test_load_documents_failures(self, mock_urlopen):
        mock_urlopen.return_value.__enter__.return_value.status = 404
        index_doc = update_disco.DocumentInfo(INDEX_1_CONTENT)
        docs = update_disco.load_documents(index_doc)
        self.assertEqual(0, len(docs))


if __name__ == "__main__":
    unittest.main()
