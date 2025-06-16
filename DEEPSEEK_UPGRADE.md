# DeepSeek LLM Service Upgrade

## Overview
This document describes the upgrade to the DeepSeek LLM service implementation to fix intermittent JSON response malformation issues.

## Changes Made

### 1. DeepSeek Go Client Integration
- **Added**: `github.com/trustsight-io/deepseek-go` v0.1.1
- **Replaced**: Manual HTTP client with official DeepSeek Go client
- **Benefits**: Built-in retry logic, proper error handling, and JSON extraction

### 2. JSON Schema Validation
- **Added**: JSON schema for recipe data validation
- **Implemented**: `JSONExtractor` with schema validation
- **Eliminated**: Manual JSON parsing and `fixDeepSeekJSON` function

### 3. Enhanced Error Handling
- **Added**: Retry logic with exponential backoff (3 retries, 2-second intervals)
- **Improved**: Logging with detailed request/response information
- **Enhanced**: Error categorization for different failure scenarios

### 4. Request Structure Updates
- **Updated**: All methods to use `deepseek.ChatCompletionRequest`
- **Added**: Proper context handling for all API calls
- **Configured**: Consistent response format settings (`json_object`)

## Environment Variables

### Required
- `DEEPSEEK_API_KEY`: DeepSeek API key
- `DEEPSEEK_API_KEY_FILE`: Alternative file-based API key (optional)

### Optional
- `DEEPSEEK_API_URL`: Custom API endpoint (defaults to DeepSeek's official endpoint)
- `DEEPSEEK_DEBUG`: Enable debug logging (`true`/`false`)

### Redis Configuration (unchanged)
- `REDIS_HOST`: Redis host (default: localhost)
- `REDIS_PORT`: Redis port (default: 6379)  
- `REDIS_PASSWORD`: Redis password
- `REDIS_DB`: Redis database number

## Breaking Changes

### API Signatures
- All LLM service methods now require proper error handling
- JSON responses are now validated against schema

### Internal Structure
- `LLMService` struct updated to use `*deepseek.Client`
- Removed `apiKey` and `apiURL` fields
- Added `jsonExtractor` field

## Migration Guide

### For Development
1. Update environment variables if using custom DeepSeek API URL
2. No code changes required for existing API consumers
3. Test recipe generation to ensure consistent JSON responses

### For Production
1. Update Docker configuration with new environment variables
2. Monitor logs for improved error reporting
3. Verify retry behavior during API rate limiting

## Testing

### Unit Tests Added
- `TestNewLLMService`: Client initialization validation
- `TestServingsType_UnmarshalJSON`: JSON unmarshaling edge cases
- Draft management operations with Redis

### Integration Testing
- All existing E2E tests should pass without modification
- Recipe generation should produce consistent, valid JSON
- Error scenarios should be handled gracefully

## Monitoring Improvements

### Logging Enhancements
- Request/response logging with `[LLMService]` prefix
- Detailed error information for troubleshooting
- JSON extraction success/failure tracking

### Metrics Recommendations
- Track successful vs failed JSON extractions
- Monitor API response times
- Count retry attempts and success rates

## Rollback Plan

If issues arise, the upgrade can be rolled back by:
1. Reverting to the previous commit
2. Removing the `github.com/trustsight-io/deepseek-go` dependency
3. Restoring the manual HTTP client implementation

## Performance Impact

### Positive
- Reduced JSON parsing errors (eliminates manual fixes)
- Built-in retry logic reduces transient failures
- Schema validation catches malformed responses early

### Considerations
- Slight increase in memory usage due to JSON schema validation
- Additional retry attempts may increase response times for failed requests
- More detailed logging may increase log volume

## Security

### Improvements
- API key validation during client initialization
- Proper context handling prevents request leaks
- Schema validation prevents malformed data injection

### Considerations
- Same API key management as before
- Debug mode should not be enabled in production
- Retry logic respects rate limiting

## Future Enhancements

1. **Caching**: Implement response caching for repeated queries
2. **Metrics**: Add Prometheus metrics for monitoring
3. **Circuit Breaker**: Implement circuit breaker pattern for API failures
4. **Streaming**: Utilize streaming responses for real-time recipe generation