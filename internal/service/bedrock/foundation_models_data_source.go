// Copyright IBM Corp. 2014, 2026
// SPDX-License-Identifier: MPL-2.0

// DONOTCOPY: Copying old resources spreads bad habits. Use skaff instead.

package bedrock

import (
	"context"

	"github.com/YakDriver/regexache"
	"github.com/aws/aws-sdk-go-v2/aws"
	"github.com/aws/aws-sdk-go-v2/service/bedrock"
	awstypes "github.com/aws/aws-sdk-go-v2/service/bedrock/types"
	"github.com/hashicorp/terraform-plugin-framework-validators/stringvalidator"
	"github.com/hashicorp/terraform-plugin-framework/datasource"
	"github.com/hashicorp/terraform-plugin-framework/datasource/schema"
	"github.com/hashicorp/terraform-plugin-framework/schema/validator"
	"github.com/hashicorp/terraform-plugin-framework/types"
	"github.com/hashicorp/terraform-provider-aws/internal/framework"
	fwflex "github.com/hashicorp/terraform-provider-aws/internal/framework/flex"
	fwtypes "github.com/hashicorp/terraform-provider-aws/internal/framework/types"
	"github.com/hashicorp/terraform-provider-aws/names"
)

// @FrameworkDataSource("aws_bedrock_foundation_models", name="Foundation Models")
func newFoundationModelsDataSource(context.Context) (datasource.DataSourceWithConfigure, error) {
	return &foundationModelsDataSource{}, nil
}

type foundationModelsDataSource struct {
	framework.DataSourceWithModel[foundationModelsDataSourceModel]
}

func (d *foundationModelsDataSource) Schema(ctx context.Context, request datasource.SchemaRequest, response *datasource.SchemaResponse) {
	response.Schema = schema.Schema{
		Attributes: map[string]schema.Attribute{
			"by_customization_type": schema.StringAttribute{
				CustomType: fwtypes.StringEnumType[awstypes.ModelCustomization](),
				Optional:   true,
			},
			"by_inference_type": schema.StringAttribute{
				CustomType: fwtypes.StringEnumType[awstypes.InferenceType](),
				Optional:   true,
			},
			"by_output_modality": schema.StringAttribute{
				CustomType: fwtypes.StringEnumType[awstypes.ModelModality](),
				Optional:   true,
			},
			"by_provider": schema.StringAttribute{
				Optional: true,
				Validators: []validator.String{
					stringvalidator.RegexMatches(regexache.MustCompile(`^[A-Za-z0-9- ]{1,63}$`), ""),
				},
			},
			names.AttrID:      framework.IDAttribute(),
			"model_summaries": framework.DataSourceComputedListOfObjectAttribute[foundationModelSummaryModel](ctx),
		},
	}
}

func (d *foundationModelsDataSource) Read(ctx context.Context, request datasource.ReadRequest, response *datasource.ReadResponse) {
	var data foundationModelsDataSourceModel
	response.Diagnostics.Append(request.Config.Get(ctx, &data)...)
	if response.Diagnostics.HasError() {
		return
	}

	conn := d.Meta().BedrockClient(ctx)

	input := &bedrock.ListFoundationModelsInput{}
	response.Diagnostics.Append(fwflex.Expand(ctx, data, input)...)
	if response.Diagnostics.HasError() {
		return
	}

	output, err := conn.ListFoundationModels(ctx, input)
	if err != nil {
		response.Diagnostics.AddError("listing Bedrock Foundation Models", err.Error())
		return
	}

	newSummaries := make([]foundationModelSummaryModel, len(output.ModelSummaries))
	for i, s := range output.ModelSummaries {
		customizations := make([]string, len(s.CustomizationsSupported))
		for j, v := range s.CustomizationsSupported {
			customizations[j] = string(v)
		}

		inferenceTypes := make([]string, len(s.InferenceTypesSupported))
		for j, v := range s.InferenceTypesSupported {
			inferenceTypes[j] = string(v)
		}

		inputModalities := make([]string, len(s.InputModalities))
		for j, v := range s.InputModalities {
			inputModalities[j] = string(v)
		}

		outputModalities := make([]string, len(s.OutputModalities))
		for j, v := range s.OutputModalities {
			outputModalities[j] = string(v)
		}

		var val fwtypes.ObjectValueOf[foundationModelLifecycleModel]

		if s.ModelLifecycle != nil {
			lifecycleModel := &foundationModelLifecycleModel{
				EndOfLifeTime:            fwflex.TimeToFramework(ctx, s.ModelLifecycle.EndOfLifeTime),
				LegacyTime:               fwflex.TimeToFramework(ctx, s.ModelLifecycle.LegacyTime),
				PublicExtendedAccessTime: fwflex.TimeToFramework(ctx, s.ModelLifecycle.PublicExtendedAccessTime),
				StartOfLifeTime:          fwflex.TimeToFramework(ctx, s.ModelLifecycle.StartOfLifeTime),
				Status:                   types.StringValue(string(s.ModelLifecycle.Status)),
			}

			v, diags := fwtypes.NewObjectValueOf[foundationModelLifecycleModel](ctx, lifecycleModel)
			response.Diagnostics.Append(diags...)
			val = v
		} else {
			val = fwtypes.NewObjectValueOfNull[foundationModelLifecycleModel](ctx)
		}

		newSummaries[i] = foundationModelSummaryModel{
			CustomizationsSupported:    fwflex.FlattenFrameworkStringValueSetOfStringLegacy(ctx, customizations),
			InferenceTypesSupported:    fwflex.FlattenFrameworkStringValueSetOfStringLegacy(ctx, inferenceTypes),
			InputModalities:            fwflex.FlattenFrameworkStringValueSetOfStringLegacy(ctx, inputModalities),
			ModelARN:                   fwtypes.ARNValue(aws.ToString(s.ModelArn)),
			ModelID:                    types.StringValue(aws.ToString(s.ModelId)),
			ModelLifecycle:             val,
			ModelName:                  types.StringValue(aws.ToString(s.ModelName)),
			OutputModalities:           fwflex.FlattenFrameworkStringValueSetOfStringLegacy(ctx, outputModalities),
			ProviderName:               types.StringValue(aws.ToString(s.ProviderName)),
			ResponseStreamingSupported: types.BoolPointerValue(s.ResponseStreamingSupported),
		}
	}

	summariesVal, fwDiags := fwtypes.NewListNestedObjectValueOfValueSlice(ctx, newSummaries)
	data.ModelSummaries = summariesVal
	response.Diagnostics.Append(fwDiags...)
	if response.Diagnostics.HasError() {
		return
	}

	data.ID = types.StringValue(d.Meta().Region(ctx))
	response.Diagnostics.Append(response.State.Set(ctx, &data)...)
}

type foundationModelsDataSourceModel struct {
	framework.WithRegionModel
	ByCustomizationType fwtypes.StringEnum[awstypes.ModelCustomization]              `tfsdk:"by_customization_type"`
	ByInferenceType     fwtypes.StringEnum[awstypes.InferenceType]                   `tfsdk:"by_inference_type"`
	ByOutputModality    fwtypes.StringEnum[awstypes.ModelModality]                   `tfsdk:"by_output_modality"`
	ByProvider          types.String                                                 `tfsdk:"by_provider"`
	ID                  types.String                                                 `tfsdk:"id"`
	ModelSummaries      fwtypes.ListNestedObjectValueOf[foundationModelSummaryModel] `tfsdk:"model_summaries"`
}

type foundationModelSummaryModel struct {
	CustomizationsSupported    fwtypes.SetOfString                                  `tfsdk:"customizations_supported"`
	InferenceTypesSupported    fwtypes.SetOfString                                  `tfsdk:"inference_types_supported"`
	InputModalities            fwtypes.SetOfString                                  `tfsdk:"input_modalities"`
	ModelARN                   fwtypes.ARN                                          `tfsdk:"model_arn"`
	ModelID                    types.String                                         `tfsdk:"model_id"`
	ModelLifecycle             fwtypes.ObjectValueOf[foundationModelLifecycleModel] `tfsdk:"model_lifecycle"`
	ModelName                  types.String                                         `tfsdk:"model_name"`
	OutputModalities           fwtypes.SetOfString                                  `tfsdk:"output_modalities"`
	ProviderName               types.String                                         `tfsdk:"provider_name"`
	ResponseStreamingSupported types.Bool                                           `tfsdk:"response_streaming_supported"`
}
