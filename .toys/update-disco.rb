# frozen_string_literal: true

# Copyright 2022 Google LLC
#
# Licensed under the Apache License, Version 2.0 (the "License");
# you may not use this file except in compliance with the License.
# You may obtain a copy of the License at
#
#     http://www.apache.org/licenses/LICENSE-2.0
#
# Unless required by applicable law or agreed to in writing, software
# distributed under the License is distributed on an "AS IS" BASIS,
# WITHOUT WARRANTIES OR CONDITIONS OF ANY KIND, either express or implied.
# See the License for the specific language governing permissions and
# limitations under the License.

desc "Update the discovery documents."

long_desc \
  "This tool downloads the latest discovery documents from the discovery " \
  "service, and updates the cache in the discovery directory. It debounces " \
  "revision changes to prevent switching to an earlier revision, and also " \
  "ignores revisions that are identical except for the revision number.",
  "",
  "By default, this tool just makes the changes locally. To open a pull " \
  "request with the changes, set the --remote= and/or --fork arguments. " \
  "To auto-approve and automerge, set the --approval-token= argument or " \
  "the APPROVAL_GITHUB_TOKEN environment variable."

flag :git_remote, "--remote=NAME" do
  desc "The name of the git remote to use as the pull request head."
end
flag :enable_fork, "--fork" do
  desc "The github user for whom to create/use a fork."
end
flag :approval_token, "--approval-token=TOKEN" do
  default ENV["APPROVAL_GITHUB_TOKEN"]
  desc "GitHub token for adding labels and approving pull requests. Defaults " \
       "to the value of the APPROVAL_GITHUB_TOKEN environment variable."
end

include :exec, e: true
include :fileutils
include "yoshi-pr-generator"

def run
  setup
  result = open_pr
  output result
end

def setup
  Dir.chdir context_directory
  @timestamp = Time.now.utc.strftime("%Y%m%d-%H%M%S")
  yoshi_utils.git_ensure_identity
  if enable_fork
    set :git_remote, "pull-request-fork" unless git_remote
    yoshi_utils.gh_ensure_fork remote: git_remote
  end
  require "net/http"
  require "json"
  require "set"
end

def open_pr
  branch_name = "disco-#{@timestamp}"
  commit_message = "chore: Automated update of discovery documents at #{@timestamp}"
  approval_message = "Rubber-stamped automated update of discovery documents!"
  yoshi_pr_generator.capture enabled: !git_remote.nil?,
                             remote: git_remote,
                             branch_name: branch_name,
                             commit_message: commit_message,
                             labels: ["automerge"],
                             auto_approve: approval_message,
                             approval_token: approval_token do
    update_disco
  end
end

class DocumentInfo
  def initialize content, filename = nil
    @filename = filename || "index.json"
    @content = content
    @json = JSON.parse content rescue nil
    @revision = @json && @json["revision"]
  end

  attr_reader :filename
  attr_reader :content
  attr_reader :json
  attr_reader :revision

  def json_without_revision
    @json_without_revision ||= begin
      result = @json.dup
      result.delete "revision"
      result.delete "etag"
      result
    end
  end
end

def update_disco
  Dir.chdir "discoveries" do
    index_document = load_index
    documents = load_documents index_document
    delete_unused_files index_document, *documents
    update_files documents
    update_index index_document
  end
end

def load_index
  logger.unknown "Loading index ..."
  content = Net::HTTP.get URI("https://discovery.googleapis.com/discovery/v1/apis") rescue nil
  document = DocumentInfo.new content
  unless document.json
    logger.fatal "Unable to load discovery index", :red, :bold
    exit 1
  end
  logger.info "Loaded discovery index"
  document
end

def load_documents index_document
  logger.unknown "Loading discovery documents ..."
  result = []
  index_document.json["items"].each do |item|
    name = item["name"]
    version = item["version"]
    discovery_rest_url = item["discoveryRestUrl"]
    filename = "#{name}.#{version}.json"
    content = Net::HTTP.get URI(discovery_rest_url) rescue nil
    document = DocumentInfo.new content, filename if content
    if document.revision
      logger.info "Loaded #{filename} from #{discovery_rest_url}"
      result << document
    else
      logger.error "Failed to load #{filename} from #{discovery_rest_url}"
    end
  end
  result
end

def delete_unused_files *documents
  expected_names = Set.new documents.map(&:filename)
  Dir.glob "*.json" do |filename|
    next if expected_names.include? filename
    logger.unknown "REMOVING file #{filename}"
    rm_rf filename
  end
end

def update_files documents
  documents.each do |document|
    filename = document.filename
    existing_content = File.read filename rescue nil
    existing = DocumentInfo.new existing_content, filename
    if existing.revision
      if existing.revision > document.revision
        logger.info "Existing revision #{existing.revision} > updated revision #{document.revision} for #{filename}"
        next
      elsif existing.revision == document.revision
        logger.info "No change to revision #{existing.revision} for #{filename}"
        next
      elsif existing.json_without_revision == document.json_without_revision
        logger.info "Revision updated #{existing.revision} to #{document.revision} but no other change for #{filename}"
        next
      end
      logger.unknown "UPDATING revision #{existing.revision} to #{document.revision} for #{filename}"
    else
      logger.unknown "WRITING new revision #{document.revision} for #{filename}"
    end
    File.write filename, document.content
  end
end

def update_index index_document
  existing_content = File.read index_document.filename
  if existing_content == index_document.content
    logger.info "Index is unchanged"
  else
    logger.unknown "UPDATING index.json"
    File.write index_document.filename, index_document.content
  end
end

def output result
  case result
  when Integer
    logger.unknown "Opened pull request #{result} for discovery documents at #{@timestamp}"
  when :unchanged
    logger.unknown "No changes for discovery documents at #{@timestamp}"
  else
    logger.unknown "Updated discovery documents at #{@timestamp} but did not open a pull request"
  end
end
