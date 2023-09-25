// Code generated by smithy-go-codegen DO NOT EDIT.

package kinesis

import (
	"context"
	"errors"
	"fmt"
	"github.com/aws/aws-sdk-go-v2/aws"
	awsmiddleware "github.com/aws/aws-sdk-go-v2/aws/middleware"
	"github.com/aws/aws-sdk-go-v2/aws/signer/v4"
	internalauth "github.com/aws/aws-sdk-go-v2/internal/auth"
	"github.com/aws/aws-sdk-go-v2/service/kinesis/types"
	smithyendpoints "github.com/aws/smithy-go/endpoints"
	"github.com/aws/smithy-go/middleware"
	"github.com/aws/smithy-go/ptr"
	smithyhttp "github.com/aws/smithy-go/transport/http"
)

// Writes a single data record into an Amazon Kinesis data stream. Call PutRecord
// to send data into the stream for real-time ingestion and subsequent processing,
// one record at a time. Each shard can support writes up to 1,000 records per
// second, up to a maximum data write total of 1 MiB per second. When invoking this
// API, it is recommended you use the StreamARN input parameter rather than the
// StreamName input parameter. You must specify the name of the stream that
// captures, stores, and transports the data; a partition key; and the data blob
// itself. The data blob can be any type of data; for example, a segment from a log
// file, geographic/location data, website clickstream data, and so on. The
// partition key is used by Kinesis Data Streams to distribute data across shards.
// Kinesis Data Streams segregates the data records that belong to a stream into
// multiple shards, using the partition key associated with each data record to
// determine the shard to which a given data record belongs. Partition keys are
// Unicode strings, with a maximum length limit of 256 characters for each key. An
// MD5 hash function is used to map partition keys to 128-bit integer values and to
// map associated data records to shards using the hash key ranges of the shards.
// You can override hashing the partition key to determine the shard by explicitly
// specifying a hash value using the ExplicitHashKey parameter. For more
// information, see Adding Data to a Stream (https://docs.aws.amazon.com/kinesis/latest/dev/developing-producers-with-sdk.html#kinesis-using-sdk-java-add-data-to-stream)
// in the Amazon Kinesis Data Streams Developer Guide. PutRecord returns the shard
// ID of where the data record was placed and the sequence number that was assigned
// to the data record. Sequence numbers increase over time and are specific to a
// shard within a stream, not across all shards within a stream. To guarantee
// strictly increasing ordering, write serially to a shard and use the
// SequenceNumberForOrdering parameter. For more information, see Adding Data to a
// Stream (https://docs.aws.amazon.com/kinesis/latest/dev/developing-producers-with-sdk.html#kinesis-using-sdk-java-add-data-to-stream)
// in the Amazon Kinesis Data Streams Developer Guide. After you write a record to
// a stream, you cannot modify that record or its order within the stream. If a
// PutRecord request cannot be processed because of insufficient provisioned
// throughput on the shard involved in the request, PutRecord throws
// ProvisionedThroughputExceededException . By default, data records are accessible
// for 24 hours from the time that they are added to a stream. You can use
// IncreaseStreamRetentionPeriod or DecreaseStreamRetentionPeriod to modify this
// retention period.
func (c *Client) PutRecord(ctx context.Context, params *PutRecordInput, optFns ...func(*Options)) (*PutRecordOutput, error) {
	if params == nil {
		params = &PutRecordInput{}
	}

	result, metadata, err := c.invokeOperation(ctx, "PutRecord", params, optFns, c.addOperationPutRecordMiddlewares)
	if err != nil {
		return nil, err
	}

	out := result.(*PutRecordOutput)
	out.ResultMetadata = metadata
	return out, nil
}

// Represents the input for PutRecord .
type PutRecordInput struct {

	// The data blob to put into the record, which is base64-encoded when the blob is
	// serialized. When the data blob (the payload before base64-encoding) is added to
	// the partition key size, the total size must not exceed the maximum record size
	// (1 MiB).
	//
	// This member is required.
	Data []byte

	// Determines which shard in the stream the data record is assigned to. Partition
	// keys are Unicode strings with a maximum length limit of 256 characters for each
	// key. Amazon Kinesis Data Streams uses the partition key as input to a hash
	// function that maps the partition key and associated data to a specific shard.
	// Specifically, an MD5 hash function is used to map partition keys to 128-bit
	// integer values and to map associated data records to shards. As a result of this
	// hashing mechanism, all data records with the same partition key map to the same
	// shard within the stream.
	//
	// This member is required.
	PartitionKey *string

	// The hash value used to explicitly determine the shard the data record is
	// assigned to by overriding the partition key hash.
	ExplicitHashKey *string

	// Guarantees strictly increasing sequence numbers, for puts from the same client
	// and to the same partition key. Usage: set the SequenceNumberForOrdering of
	// record n to the sequence number of record n-1 (as returned in the result when
	// putting record n-1). If this parameter is not set, records are coarsely ordered
	// based on arrival time.
	SequenceNumberForOrdering *string

	// The ARN of the stream.
	StreamARN *string

	// The name of the stream to put the data record into.
	StreamName *string

	noSmithyDocumentSerde
}

// Represents the output for PutRecord .
type PutRecordOutput struct {

	// The sequence number identifier that was assigned to the put data record. The
	// sequence number for the record is unique across all records in the stream. A
	// sequence number is the identifier associated with every record put into the
	// stream.
	//
	// This member is required.
	SequenceNumber *string

	// The shard ID of the shard where the data record was placed.
	//
	// This member is required.
	ShardId *string

	// The encryption type to use on the record. This parameter can be one of the
	// following values:
	//   - NONE : Do not encrypt the records in the stream.
	//   - KMS : Use server-side encryption on the records in the stream using a
	//   customer-managed Amazon Web Services KMS key.
	EncryptionType types.EncryptionType

	// Metadata pertaining to the operation's result.
	ResultMetadata middleware.Metadata

	noSmithyDocumentSerde
}

func (c *Client) addOperationPutRecordMiddlewares(stack *middleware.Stack, options Options) (err error) {
	err = stack.Serialize.Add(&awsAwsjson11_serializeOpPutRecord{}, middleware.After)
	if err != nil {
		return err
	}
	err = stack.Deserialize.Add(&awsAwsjson11_deserializeOpPutRecord{}, middleware.After)
	if err != nil {
		return err
	}
	if err = addlegacyEndpointContextSetter(stack, options); err != nil {
		return err
	}
	if err = addSetLoggerMiddleware(stack, options); err != nil {
		return err
	}
	if err = awsmiddleware.AddClientRequestIDMiddleware(stack); err != nil {
		return err
	}
	if err = smithyhttp.AddComputeContentLengthMiddleware(stack); err != nil {
		return err
	}
	if err = addResolveEndpointMiddleware(stack, options); err != nil {
		return err
	}
	if err = v4.AddComputePayloadSHA256Middleware(stack); err != nil {
		return err
	}
	if err = addRetryMiddlewares(stack, options); err != nil {
		return err
	}
	if err = addHTTPSignerV4Middleware(stack, options); err != nil {
		return err
	}
	if err = awsmiddleware.AddRawResponseToMetadata(stack); err != nil {
		return err
	}
	if err = awsmiddleware.AddRecordResponseTiming(stack); err != nil {
		return err
	}
	if err = addClientUserAgent(stack, options); err != nil {
		return err
	}
	if err = smithyhttp.AddErrorCloseResponseBodyMiddleware(stack); err != nil {
		return err
	}
	if err = smithyhttp.AddCloseResponseBodyMiddleware(stack); err != nil {
		return err
	}
	if err = addPutRecordResolveEndpointMiddleware(stack, options); err != nil {
		return err
	}
	if err = addOpPutRecordValidationMiddleware(stack); err != nil {
		return err
	}
	if err = stack.Initialize.Add(newServiceMetadataMiddleware_opPutRecord(options.Region), middleware.Before); err != nil {
		return err
	}
	if err = awsmiddleware.AddRecursionDetection(stack); err != nil {
		return err
	}
	if err = addRequestIDRetrieverMiddleware(stack); err != nil {
		return err
	}
	if err = addResponseErrorMiddleware(stack); err != nil {
		return err
	}
	if err = addRequestResponseLogging(stack, options); err != nil {
		return err
	}
	if err = addendpointDisableHTTPSMiddleware(stack, options); err != nil {
		return err
	}
	return nil
}

func newServiceMetadataMiddleware_opPutRecord(region string) *awsmiddleware.RegisterServiceMetadata {
	return &awsmiddleware.RegisterServiceMetadata{
		Region:        region,
		ServiceID:     ServiceID,
		SigningName:   "kinesis",
		OperationName: "PutRecord",
	}
}

type opPutRecordResolveEndpointMiddleware struct {
	EndpointResolver EndpointResolverV2
	BuiltInResolver  builtInParameterResolver
}

func (*opPutRecordResolveEndpointMiddleware) ID() string {
	return "ResolveEndpointV2"
}

func (m *opPutRecordResolveEndpointMiddleware) HandleSerialize(ctx context.Context, in middleware.SerializeInput, next middleware.SerializeHandler) (
	out middleware.SerializeOutput, metadata middleware.Metadata, err error,
) {
	if awsmiddleware.GetRequiresLegacyEndpoints(ctx) {
		return next.HandleSerialize(ctx, in)
	}

	req, ok := in.Request.(*smithyhttp.Request)
	if !ok {
		return out, metadata, fmt.Errorf("unknown transport type %T", in.Request)
	}

	input, ok := in.Parameters.(*PutRecordInput)
	if !ok {
		return out, metadata, fmt.Errorf("unknown transport type %T", in.Request)
	}

	if m.EndpointResolver == nil {
		return out, metadata, fmt.Errorf("expected endpoint resolver to not be nil")
	}

	params := EndpointParameters{}

	m.BuiltInResolver.ResolveBuiltIns(&params)

	params.StreamARN = input.StreamARN

	params.OperationType = ptr.String("data")

	var resolvedEndpoint smithyendpoints.Endpoint
	resolvedEndpoint, err = m.EndpointResolver.ResolveEndpoint(ctx, params)
	if err != nil {
		return out, metadata, fmt.Errorf("failed to resolve service endpoint, %w", err)
	}

	req.URL = &resolvedEndpoint.URI

	for k := range resolvedEndpoint.Headers {
		req.Header.Set(
			k,
			resolvedEndpoint.Headers.Get(k),
		)
	}

	authSchemes, err := internalauth.GetAuthenticationSchemes(&resolvedEndpoint.Properties)
	if err != nil {
		var nfe *internalauth.NoAuthenticationSchemesFoundError
		if errors.As(err, &nfe) {
			// if no auth scheme is found, default to sigv4
			signingName := "kinesis"
			signingRegion := m.BuiltInResolver.(*builtInResolver).Region
			ctx = awsmiddleware.SetSigningName(ctx, signingName)
			ctx = awsmiddleware.SetSigningRegion(ctx, signingRegion)

		}
		var ue *internalauth.UnSupportedAuthenticationSchemeSpecifiedError
		if errors.As(err, &ue) {
			return out, metadata, fmt.Errorf(
				"This operation requests signer version(s) %v but the client only supports %v",
				ue.UnsupportedSchemes,
				internalauth.SupportedSchemes,
			)
		}
	}

	for _, authScheme := range authSchemes {
		switch authScheme.(type) {
		case *internalauth.AuthenticationSchemeV4:
			v4Scheme, _ := authScheme.(*internalauth.AuthenticationSchemeV4)
			var signingName, signingRegion string
			if v4Scheme.SigningName == nil {
				signingName = "kinesis"
			} else {
				signingName = *v4Scheme.SigningName
			}
			if v4Scheme.SigningRegion == nil {
				signingRegion = m.BuiltInResolver.(*builtInResolver).Region
			} else {
				signingRegion = *v4Scheme.SigningRegion
			}
			if v4Scheme.DisableDoubleEncoding != nil {
				// The signer sets an equivalent value at client initialization time.
				// Setting this context value will cause the signer to extract it
				// and override the value set at client initialization time.
				ctx = internalauth.SetDisableDoubleEncoding(ctx, *v4Scheme.DisableDoubleEncoding)
			}
			ctx = awsmiddleware.SetSigningName(ctx, signingName)
			ctx = awsmiddleware.SetSigningRegion(ctx, signingRegion)
			break
		case *internalauth.AuthenticationSchemeV4A:
			v4aScheme, _ := authScheme.(*internalauth.AuthenticationSchemeV4A)
			if v4aScheme.SigningName == nil {
				v4aScheme.SigningName = aws.String("kinesis")
			}
			if v4aScheme.DisableDoubleEncoding != nil {
				// The signer sets an equivalent value at client initialization time.
				// Setting this context value will cause the signer to extract it
				// and override the value set at client initialization time.
				ctx = internalauth.SetDisableDoubleEncoding(ctx, *v4aScheme.DisableDoubleEncoding)
			}
			ctx = awsmiddleware.SetSigningName(ctx, *v4aScheme.SigningName)
			ctx = awsmiddleware.SetSigningRegion(ctx, v4aScheme.SigningRegionSet[0])
			break
		case *internalauth.AuthenticationSchemeNone:
			break
		}
	}

	return next.HandleSerialize(ctx, in)
}

func addPutRecordResolveEndpointMiddleware(stack *middleware.Stack, options Options) error {
	return stack.Serialize.Insert(&opPutRecordResolveEndpointMiddleware{
		EndpointResolver: options.EndpointResolverV2,
		BuiltInResolver: &builtInResolver{
			Region:       options.Region,
			UseDualStack: options.EndpointOptions.UseDualStackEndpoint,
			UseFIPS:      options.EndpointOptions.UseFIPSEndpoint,
			Endpoint:     options.BaseEndpoint,
		},
	}, "ResolveEndpoint", middleware.After)
}
