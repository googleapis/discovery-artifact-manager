/* Copyright 2016 Google Inc
 *
 * Licensed under the Apache License, Version 2.0 (the "License");
 * you may not use this file except in compliance with the License.
 * You may obtain a copy of the License at
 *
 *      http://www.apache.org/licenses/LICENSE-2.0
 *
 * Unless required by applicable law or agreed to in writing, software
 * distributed under the License is distributed on an "AS IS" BASIS,
 * WITHOUT WARRANTIES OR CONDITIONS OF ANY KIND, either express or implied.
 * See the License for the specific language governing permissions and
 * limitations under the License.
 */
package com.google.api.codegen.transformer.nodejs;

import com.google.api.codegen.GeneratorVersionProvider;
import com.google.api.codegen.ServiceMessages;
import com.google.api.codegen.TargetLanguage;
import com.google.api.codegen.config.ApiModel;
import com.google.api.codegen.config.GapicProductConfig;
import com.google.api.codegen.config.GrpcStreamingConfig;
import com.google.api.codegen.config.InterfaceModel;
import com.google.api.codegen.config.LongRunningConfig;
import com.google.api.codegen.config.MethodModel;
import com.google.api.codegen.config.PackageMetadataConfig;
import com.google.api.codegen.config.ProtoApiModel;
import com.google.api.codegen.config.VisibilityConfig;
import com.google.api.codegen.gapic.GapicCodePathMapper;
import com.google.api.codegen.nodejs.NodeJSUtils;
import com.google.api.codegen.transformer.BatchingTransformer;
import com.google.api.codegen.transformer.DynamicLangApiMethodTransformer;
import com.google.api.codegen.transformer.FileHeaderTransformer;
import com.google.api.codegen.transformer.GapicInterfaceContext;
import com.google.api.codegen.transformer.GapicMethodContext;
import com.google.api.codegen.transformer.GrpcStubTransformer;
import com.google.api.codegen.transformer.ModelToViewTransformer;
import com.google.api.codegen.transformer.ModelTypeTable;
import com.google.api.codegen.transformer.PageStreamingTransformer;
import com.google.api.codegen.transformer.PathTemplateTransformer;
import com.google.api.codegen.transformer.ServiceTransformer;
import com.google.api.codegen.transformer.SurfaceNamer;
import com.google.api.codegen.util.Name;
import com.google.api.codegen.util.js.JSTypeTable;
import com.google.api.codegen.viewmodel.ApiMethodView;
import com.google.api.codegen.viewmodel.DynamicLangXApiView;
import com.google.api.codegen.viewmodel.GrpcStreamingDetailView;
import com.google.api.codegen.viewmodel.GrpcStubView;
import com.google.api.codegen.viewmodel.ImportSectionView;
import com.google.api.codegen.viewmodel.LongRunningOperationDetailView;
import com.google.api.codegen.viewmodel.PathTemplateGetterFunctionView;
import com.google.api.codegen.viewmodel.ViewModel;
import com.google.api.codegen.viewmodel.metadata.VersionIndexRequireView;
import com.google.api.codegen.viewmodel.metadata.VersionIndexView;
import com.google.api.tools.framework.model.Model;
import com.google.api.tools.framework.model.TypeRef;
import com.google.common.collect.ImmutableList;
import com.google.common.collect.Iterables;
import java.util.ArrayList;
import java.util.Comparator;
import java.util.List;
import java.util.Set;
import java.util.TreeSet;

/** Responsible for producing GAPIC surface views for NodeJS */
public class NodeJSGapicSurfaceTransformer implements ModelToViewTransformer {
  private static final String INDEX_TEMPLATE_FILE = "nodejs/index.snip";
  private static final String VERSION_INDEX_TEMPLATE_FILE = "nodejs/version_index.snip";
  private static final String XAPI_TEMPLATE_FILENAME = "nodejs/main.snip";

  private final GapicCodePathMapper pathMapper;
  private final FileHeaderTransformer fileHeaderTransformer =
      new FileHeaderTransformer(new NodeJSImportSectionTransformer());
  private final DynamicLangApiMethodTransformer apiMethodTransformer =
      new DynamicLangApiMethodTransformer(new NodeJSApiMethodParamTransformer());
  private final ServiceTransformer serviceTransformer = new ServiceTransformer();
  private final GrpcStubTransformer grpcStubTransformer = new GrpcStubTransformer();
  private final PageStreamingTransformer pageStreamingTransformer = new PageStreamingTransformer();
  private final BatchingTransformer batchingTransformer = new BatchingTransformer();
  private final PathTemplateTransformer pathTemplateTransformer = new PathTemplateTransformer();
  private final PackageMetadataConfig packageConfig;

  public NodeJSGapicSurfaceTransformer(
      GapicCodePathMapper pathMapper, PackageMetadataConfig packageConfig) {
    this.pathMapper = pathMapper;
    this.packageConfig = packageConfig;
  }

  @Override
  public List<String> getTemplateFileNames() {
    return ImmutableList.of(
        INDEX_TEMPLATE_FILE, VERSION_INDEX_TEMPLATE_FILE, XAPI_TEMPLATE_FILENAME);
  }

  @Override
  public List<ViewModel> transform(Model model, GapicProductConfig productConfig) {
    ApiModel apiModel = new ProtoApiModel(model);
    Iterable<? extends InterfaceModel> apiInterfaces = apiModel.getInterfaces(productConfig);
    ImmutableList.Builder<ViewModel> models = ImmutableList.builder();
    models.addAll(generateIndexViews(apiInterfaces, productConfig));
    models.addAll(generateApiClasses(apiModel, productConfig));
    return models.build();
  }

  private List<ViewModel> generateApiClasses(ApiModel model, GapicProductConfig productConfig) {
    ImmutableList.Builder<ViewModel> models = ImmutableList.builder();
    Iterable<? extends InterfaceModel> interfaces = model.getInterfaces(productConfig);
    boolean hasMultipleServices = Iterables.size(interfaces) > 1;
    for (InterfaceModel apiInterface : interfaces) {
      GapicInterfaceContext context = createContext(apiInterface, productConfig);
      models.add(generateApiClass(context, hasMultipleServices));
    }
    return models.build();
  }

  private ViewModel generateApiClass(GapicInterfaceContext context, boolean hasMultipleServices) {
    SurfaceNamer namer = context.getNamer();
    String subPath =
        pathMapper.getOutputPath(context.getInterface().getFullName(), context.getProductConfig());
    List<ApiMethodView> methods = generateApiMethods(context, hasMultipleServices);

    DynamicLangXApiView.Builder xapiClass = DynamicLangXApiView.newBuilder();

    xapiClass.templateFileName(XAPI_TEMPLATE_FILENAME);
    xapiClass.outputPath(subPath + "/" + namer.getServiceFileName(context.getInterfaceConfig()));

    xapiClass.fileHeader(fileHeaderTransformer.generateFileHeader(context));
    xapiClass.protoFilename(context.getInterface().getFile().getSimpleName());

    xapiClass.name(namer.getApiWrapperClassName(context.getInterfaceConfig()));
    xapiClass.doc(serviceTransformer.generateServiceDoc(context, methods.get(0)));
    xapiClass.stubs(grpcStubTransformer.generateGrpcStubs(context));

    ApiModel model = context.getApiModel();
    xapiClass.serviceAddress(model.getServiceAddress());
    xapiClass.servicePort(model.getServicePort());
    xapiClass.serviceTitle(model.getTitle());
    xapiClass.authScopes(model.getAuthScopes());
    xapiClass.hasDefaultServiceAddress(context.getInterfaceConfig().hasDefaultServiceAddress());
    xapiClass.hasDefaultServiceScopes(context.getInterfaceConfig().hasDefaultServiceScopes());

    xapiClass.pageStreamingDescriptors(pageStreamingTransformer.generateDescriptors(context));
    xapiClass.batchingDescriptors(batchingTransformer.generateDescriptors(context));
    xapiClass.longRunningDescriptors(createLongRunningDescriptors(context));
    xapiClass.grpcStreamingDescriptors(createGrpcStreamingDescriptors(context));
    xapiClass.hasPageStreamingMethods(context.getInterfaceConfig().hasPageStreamingMethods());
    xapiClass.hasBatchingMethods(context.getInterfaceConfig().hasBatchingMethods());
    xapiClass.hasLongRunningOperations(context.getInterfaceConfig().hasLongRunningOperations());

    xapiClass.pathTemplates(pathTemplateTransformer.generatePathTemplates(context));
    xapiClass.formatResourceFunctions(
        pathTemplateTransformer.generateFormatResourceFunctions(context));
    xapiClass.parseResourceFunctions(
        pathTemplateTransformer.generateParseResourceFunctions(context));
    xapiClass.pathTemplateGetterFunctions(ImmutableList.<PathTemplateGetterFunctionView>of());

    xapiClass.methodKeys(ImmutableList.<String>of());
    xapiClass.interfaceKey(context.getInterface().getFullName());
    xapiClass.clientConfigPath(namer.getClientConfigPath(context.getInterfaceModel()));
    xapiClass.grpcClientTypeName(
        namer.getAndSaveNicknameForGrpcClientTypeName(
            context.getImportTypeTable(), context.getInterfaceModel()));

    xapiClass.apiMethods(methods);

    xapiClass.toolkitVersion(GeneratorVersionProvider.getGeneratorVersion());

    xapiClass.packageVersion(
        packageConfig.generatedPackageVersionBound(TargetLanguage.NODEJS).lower());

    xapiClass.apiVersion(packageConfig.apiVersion());

    xapiClass.packageHasMultipleServices(hasMultipleServices);
    xapiClass.packageServiceName(namer.getPackageServiceName(context.getInterfaceModel()));

    xapiClass.validDescriptorsNames(generateValidDescriptorsNames(context));
    xapiClass.constructorName(namer.getApiWrapperClassConstructorName(context.getInterfaceModel()));
    xapiClass.isGcloud(NodeJSUtils.isGcloud(context.getProductConfig()));

    return xapiClass.build();
  }

  private List<String> generateValidDescriptorsNames(GapicInterfaceContext context) {
    ImmutableList.Builder<String> validDescriptorsNames = ImmutableList.builder();
    if (context.getInterfaceConfig().hasPageStreamingMethods()) {
      validDescriptorsNames.add("this._descriptors.page");
    }
    if (context.getInterfaceConfig().hasBatchingMethods()) {
      validDescriptorsNames.add("this._descriptors.batching");
    }
    if (context.getInterfaceConfig().hasGrpcStreamingMethods()) {
      validDescriptorsNames.add("this._descriptors.stream");
    }
    if (context.getInterfaceConfig().hasLongRunningOperations()) {
      validDescriptorsNames.add("this._descriptors.longrunning");
    }
    return validDescriptorsNames.build();
  }

  private List<ApiMethodView> generateApiMethods(
      GapicInterfaceContext context, boolean packageHasMultipleServices) {
    ImmutableList.Builder<ApiMethodView> apiMethods = ImmutableList.builder();

    for (MethodModel method : context.getSupportedMethods()) {
      apiMethods.add(
          apiMethodTransformer.generateMethod(
              context.asDynamicMethodContext(method), packageHasMultipleServices));
    }

    return apiMethods.build();
  }

  private List<GrpcStreamingDetailView> createGrpcStreamingDescriptors(
      GapicInterfaceContext context) {
    List<GrpcStreamingDetailView> result = new ArrayList<>();

    for (MethodModel method : context.getGrpcStreamingMethods()) {
      GrpcStreamingConfig grpcStreamingConfig =
          context.asDynamicMethodContext(method).getMethodConfig().getGrpcStreaming();
      String resourcesFieldGetFunction = null;
      if (grpcStreamingConfig.hasResourceField()) {
        resourcesFieldGetFunction =
            context.getNamer().getFieldGetFunctionName(grpcStreamingConfig.getResourcesField());
      }

      result.add(
          GrpcStreamingDetailView.newBuilder()
              .methodName(context.getNamer().getApiMethodName(method, VisibilityConfig.PUBLIC))
              .grpcStreamingType(grpcStreamingConfig.getType())
              .grpcResourcesField(resourcesFieldGetFunction)
              .streamTypeName(context.getNamer().getStreamTypeName(grpcStreamingConfig.getType()))
              .build());
    }

    return result;
  }

  private List<LongRunningOperationDetailView> createLongRunningDescriptors(
      GapicInterfaceContext context) {
    List<LongRunningOperationDetailView> result = new ArrayList<>();

    for (MethodModel method : context.getLongRunningMethods()) {
      GapicMethodContext methodContext = context.asDynamicMethodContext(method);
      LongRunningConfig lroConfig = methodContext.getMethodConfig().getLongRunningConfig();
      TypeRef returnType = lroConfig.getReturnType();
      TypeRef metadataType = lroConfig.getMetadataType();
      result.add(
          LongRunningOperationDetailView.newBuilder()
              .methodName(context.getNamer().getApiMethodName(method, VisibilityConfig.PUBLIC))
              .constructorName("")
              .clientReturnTypeName("")
              .operationPayloadTypeName(context.getImportTypeTable().getFullNameFor(returnType))
              .isEmptyOperation(ServiceMessages.s_isEmptyType(lroConfig.getReturnType()))
              .metadataTypeName(context.getImportTypeTable().getFullNameFor(metadataType))
              .implementsCancel(true)
              .implementsDelete(true)
              .initialPollDelay(lroConfig.getInitialPollDelay().toMillis())
              .pollDelayMultiplier(lroConfig.getPollDelayMultiplier())
              .maxPollDelay(lroConfig.getMaxPollDelay().toMillis())
              .totalPollTimeout(lroConfig.getTotalPollTimeout().toMillis())
              .build());
    }

    return result;
  }

  private List<ViewModel> generateIndexViews(
      Iterable<? extends InterfaceModel> apiInterfaces, GapicProductConfig productConfig) {
    NodeJSPackageMetadataNamer packageMetadataNamer =
        new NodeJSPackageMetadataNamer(
            productConfig.getPackageName(), productConfig.getDomainLayerLocation());
    ArrayList<ViewModel> indexViews = new ArrayList<>();
    NodeJSSurfaceNamer namer =
        new NodeJSSurfaceNamer(productConfig.getPackageName(), NodeJSUtils.isGcloud(productConfig));
    String version = namer.getApiWrapperModuleVersion();
    boolean hasVersion = version != null && !version.isEmpty();
    ArrayList<VersionIndexRequireView> requireViews = new ArrayList<>();
    boolean packageHasMultipleServices = Iterables.size(apiInterfaces) > 1;
    for (InterfaceModel apiInterface : apiInterfaces) {
      Name serviceName = namer.getReducedServiceName(apiInterface.getSimpleName());
      String localName =
          hasVersion ? serviceName.join(version).toLowerCamel() : serviceName.toLowerCamel();
      GapicInterfaceContext context = createContext(apiInterface, productConfig);
      VersionIndexRequireView require =
          VersionIndexRequireView.newBuilder()
              .clientName(
                  namer.getApiWrapperClassName(productConfig.getInterfaceConfig(apiInterface)))
              .serviceName(namer.getPackageServiceName(apiInterface))
              .localName(localName)
              .doc(
                  serviceTransformer.generateServiceDoc(
                      context, generateApiMethods(context, packageHasMultipleServices).get(0)))
              .fileName(namer.getClientFileName(context.getInterfaceModel()))
              .build();
      requireViews.add(require);
    }
    String indexOutputPath = hasVersion ? "src/" + version + "/index.js" : "src/index.js";
    VersionIndexView.Builder indexViewbuilder =
        VersionIndexView.newBuilder()
            .templateFileName(INDEX_TEMPLATE_FILE)
            .outputPath(indexOutputPath)
            .requireViews(requireViews)
            .primaryService(requireViews.get(0))
            .packageVersion(
                packageConfig.generatedPackageVersionBound(TargetLanguage.NODEJS).lower())
            .toolkitVersion(GeneratorVersionProvider.getGeneratorVersion())
            .fileHeader(
                fileHeaderTransformer.generateFileHeader(
                    productConfig, ImportSectionView.newBuilder().build(), namer))
            .packageName(packageMetadataNamer.getMetadataIdentifier());
    if (hasVersion) {
      indexViewbuilder.apiVersion(version);
    }
    indexViews.add(indexViewbuilder.build());

    if (hasVersion) {
      String versionIndexOutputPath = "src/index.js";
      VersionIndexView.Builder versionIndexViewBuilder =
          VersionIndexView.newBuilder()
              .templateFileName(VERSION_INDEX_TEMPLATE_FILE)
              .outputPath(versionIndexOutputPath)
              .requireViews(requireViews)
              .primaryService(requireViews.get(0))
              .apiVersion(version)
              .stubs(versionIndexStubs(apiInterfaces, productConfig))
              .isGcloud(NodeJSUtils.isGcloud(productConfig))
              .packageVersion(
                  packageConfig.generatedPackageVersionBound(TargetLanguage.NODEJS).lower())
              .toolkitVersion(GeneratorVersionProvider.getGeneratorVersion())
              .fileHeader(
                  fileHeaderTransformer.generateFileHeader(
                      productConfig, ImportSectionView.newBuilder().build(), namer))
              .packageName(packageMetadataNamer.getMetadataIdentifier())
              .namespace(packageMetadataNamer.getServiceName());
      indexViews.add(versionIndexViewBuilder.build());
    }
    return indexViews;
  }

  private List<GrpcStubView> versionIndexStubs(
      Iterable<? extends InterfaceModel> apiInterfaces, GapicProductConfig productConfig) {
    Set<GrpcStubView> stubs = new TreeSet<>(stubViewComparator());
    for (InterfaceModel apiInterface : apiInterfaces) {
      stubs.addAll(
          grpcStubTransformer.generateGrpcStubs(createContext(apiInterface, productConfig)));
    }
    return ImmutableList.copyOf(stubs);
  }

  private Comparator<GrpcStubView> stubViewComparator() {
    return new Comparator<GrpcStubView>() {
      @Override
      public int compare(GrpcStubView o1, GrpcStubView o2) {
        return o1.protoFileName().compareTo(o2.protoFileName());
      }
    };
  }

  private GapicInterfaceContext createContext(
      InterfaceModel apiInterface, GapicProductConfig productConfig) {
    return GapicInterfaceContext.create(
        apiInterface,
        productConfig,
        new ModelTypeTable(
            new JSTypeTable(productConfig.getPackageName()),
            new NodeJSModelTypeNameConverter(productConfig.getPackageName())),
        new NodeJSSurfaceNamer(productConfig.getPackageName(), NodeJSUtils.isGcloud(productConfig)),
        new NodeJSFeatureConfig());
  }
}
