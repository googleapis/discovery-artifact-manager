<?php
/*
 * Copyright 2016 Google Inc.
 *
 * Licensed under the Apache License, Version 2.0 (the "License"); you may not
 * use this file except in compliance with the License. You may obtain a copy of
 * the License at
 *
 * http://www.apache.org/licenses/LICENSE-2.0
 *
 * Unless required by applicable law or agreed to in writing, software
 * distributed under the License is distributed on an "AS IS" BASIS, WITHOUT
 * WARRANTIES OR CONDITIONS OF ANY KIND, either express or implied. See the
 * License for the specific language governing permissions and limitations under
 * the License.
 */

class Google_Service_Slides_Page extends Google_Collection
{
  protected $collection_key = 'pageElements';
  protected $layoutPropertiesType = 'Google_Service_Slides_LayoutProperties';
  protected $layoutPropertiesDataType = '';
  public $objectId;
  protected $pageElementsType = 'Google_Service_Slides_PageElement';
  protected $pageElementsDataType = 'array';
  protected $pagePropertiesType = 'Google_Service_Slides_PageProperties';
  protected $pagePropertiesDataType = '';
  public $pageType;
  protected $slidePropertiesType = 'Google_Service_Slides_SlideProperties';
  protected $slidePropertiesDataType = '';

  public function setLayoutProperties(Google_Service_Slides_LayoutProperties $layoutProperties)
  {
    $this->layoutProperties = $layoutProperties;
  }
  public function getLayoutProperties()
  {
    return $this->layoutProperties;
  }
  public function setObjectId($objectId)
  {
    $this->objectId = $objectId;
  }
  public function getObjectId()
  {
    return $this->objectId;
  }
  public function setPageElements($pageElements)
  {
    $this->pageElements = $pageElements;
  }
  public function getPageElements()
  {
    return $this->pageElements;
  }
  public function setPageProperties(Google_Service_Slides_PageProperties $pageProperties)
  {
    $this->pageProperties = $pageProperties;
  }
  public function getPageProperties()
  {
    return $this->pageProperties;
  }
  public function setPageType($pageType)
  {
    $this->pageType = $pageType;
  }
  public function getPageType()
  {
    return $this->pageType;
  }
  public function setSlideProperties(Google_Service_Slides_SlideProperties $slideProperties)
  {
    $this->slideProperties = $slideProperties;
  }
  public function getSlideProperties()
  {
    return $this->slideProperties;
  }
}
