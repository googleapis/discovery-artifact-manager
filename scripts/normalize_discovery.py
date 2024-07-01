# Copyright 2024 Google LLC
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

"""Updates the discovery documents in the discoveries directory."""

import glob
import json
import logging
import os


def main() -> None:
    """The entrypoint for normalizing discovery documents.

    This script sorts the JSON keys in each discovery document and writes it back
    to the file. This makes it easier to compare documents for semantic diffs.
    """
    logging.basicConfig(level=logging.INFO)
    os.chdir("discoveries")
    for filename in glob.glob("*.json"):
        logging.info("Normalizing %s", filename)
        with open(filename, "rb+") as f:
            content = f.read()
            data = json.loads(content)
            normalized_content = json.dumps(data, indent=2, sort_keys=True).encode(
                "utf-8"
            )

        with open(filename, "wb") as f:
            f.write(normalized_content)
    os.chdir("..")


if __name__ == "__main__":
    main()
