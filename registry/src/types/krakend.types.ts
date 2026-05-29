export interface KrakendConfig {
  $schema?: 'https://www.krakend.io/schema/v2.12/krakend.json';
  async_agent?: KrakendAsyncAgent;
  cache_ttl?: string;
  client_tls?: object; // ToDo
  debug_endpoint?: boolean;
  dialer_fallback_delay?: string;
  dialer_keep_alive?: string;
  dialer_timeout?: string;
  disable_compression?: boolean;
  disable_keep_alives?: boolean;
  disable_rest?: boolean;
  dns_cache_ttl?: string;
  echo_endpoint?: boolean;
  endpoints: KrakendEndpoint[];
  expect_continue_timeout?: string;
  extra_config?: KrakendServiceExtraConfig;
  host?: string[];
  idle_connection_timeout?: string;
  idle_timeout?: string;
  listen_ip?: string;
  max_header_bytes?: number;
  max_idle_connections?: number;
  max_idle_connections_per_host?: number;
  max_shutdown_wait_time?: string;
  name?: string;
  output_encoding?:
    | 'json'
    | 'fast-json'
    | 'json-collection'
    | 'xml'
    | 'negotiate'
    | 'string'
    | 'no-op';
  plugin?: {
    pattern: string;
    folder: string;
  };
  port?: number;
  read_header_timeout?: string;
  read_timeout?: string;
  response_header_timeout?: string;
  sequential_start?: boolean;
  timeout?: string;
  tls?: object; // ToDo
  use_h2c?: boolean;
  version: number;
  write_timeout?: string;
}

export interface KrakendServiceExtraConfig {
  'ai/mcp'?: object; // ToDo
  'auth/revoker'?: object; // ToDo
  'auth/validator'?: KrakendAuthValidator;
  'governance/processors'?: object; // ToDo
  'governance/quota'?: object; // ToDo
  'modifier/lua-endpoint'?: KrakendModifierLua;
  'plugin/http-server'?: object; // ToDo
  redis?: object; // ToDo
  router?: object; // ToDo
  'security/bot-detector'?: KrakendSecurityBotDetector;
  'security/cors'?: KrakendSecurityCors;
  'security/http'?: KrakendSecurityHttp;
  'server/static-filesystem'?: object; // ToDo
  'server/virtualhost'?: object; // ToDo
  'telemetry/gelf'?: object; // ToDo
  'telemetry/influx'?: object; // ToDo
  'telemetry/logging'?: object; // ToDo
  'telemetry/logstash'?: object; // ToDo
  'telemetry/metrics'?: object; // ToDo
  'telemetry/moesif'?: object; // ToDo
  'telemetry/newrelic'?: object; // ToDo
  'telemetry/opencensus'?: object; // ToDo
  'telemetry/opentelemetry'?: object; // ToDo
  'telemetry/opentelemetry-security'?: object; // ToDo
}

export enum KrakendHttpMethod {
  GET = 'GET',
  POST = 'POST',
  PUT = 'PUT',
  PATCH = 'PATCH',
  DELETE = 'DELETE',
}

export enum KrakendEncoding {
  json = 'json',
  'json-collection' = 'json-collection',
  yaml = 'yaml',
  'fast-json' = 'fast-json',
  xml = 'xml',
  negotiate = 'negotiate',
  string = 'string',
  'no-op' = 'no-op',
}

export enum KrakendAuthAlg {
  EdDSA = 'EdDSA',
  HS256 = 'HS256',
  HS384 = 'HS384',
  HS512 = 'HS512',
  RS256 = 'RS256',
  RS384 = 'RS384',
  RS512 = 'RS512',
  ES256 = 'ES256',
  ES384 = 'ES384',
  ES512 = 'ES512',
  PS256 = 'PS256',
  PS384 = 'PS384',
  PS512 = 'PS512',
}

export type KrakendCipherSuite =
  | 5
  | 10
  | 47
  | 53
  | 60
  | 156
  | 157
  | 49159
  | 49161
  | 49162
  | 49169
  | 49170
  | 49171
  | 49172
  | 49187
  | 49191
  | 49199
  | 49195
  | 49200
  | 49196
  | 52392
  | 52393;

export interface KrakendAuthSigner {
  alg: KrakendAuthAlg;
  cipher_suites?: KrakendCipherSuite[];
  cypher_key?: string;
  disable_jwk_security?: boolean;
  full?: boolean;
  jwk_fingerprints?: string[];
  jwk_local_ca?: string;
  jwk_local_path?: string;
  jwk_url?: string;
  keys_to_sign: string[];
  kid: string;
  leeway?: string;
  secret_url?: string;
}

export interface KrakendAuthValidator {
  alg?: KrakendAuthAlg;
  audience?: string[];
  auth_header_name?: string;
  cache?: boolean;
  cache_duration?: number;
  cipher_suites?: KrakendCipherSuite[];
  cookie_key?: string;
  cypher_key?: string;
  disable_jwk_security?: boolean;
  failed_jwk_key_cooldown?: string;
  issuer?: string;
  jwk_fingerprints?: string[];
  jwk_local_ca?: string;
  jwk_local_path?: string;
  jwk_url?: string;
  key_identify_strategy?: 'kid' | 'x5t' | 'x5t#S256' | 'kid_x5t';
  leeway?: string;
  operation_debug?: boolean;
  propagate_claims?: string[][];
  propagate_claims_preserve_array?: boolean;
  roles?: string[];
  roles_key?: string;
  roles_key_is_nested?: boolean;
  scopes?: string;
  scopes_key?: string;
  scopes_matcher?: 'any' | 'all';
  secret_url?: string;
}

export interface KrakendModifierLua {
  allow_open_libs?: boolean;
  live?: boolean;
  md5: Record<string, string>[];
  post?: string;
  pre?: string;
  skip_next?: boolean;
  sources?: string[];
}

export interface KrakendQosRatelimitProxy {
  capacity: number;
  every?: string;
  max_rate: number;
}

export interface KrakendQosRateLimitRouter {
  capacity?: number;
  cleanup_period?: string;
  cleanup_threads?: number;
  client_capacity?: number;
  client_max_rate?: number;
  every?: string;
  key?: string;
  max_rate?: number;
  num_shards?: number;
  strategy?: 'ip' | 'header' | 'param';
}

export interface KrakendSecurityBotDetector {
  allow?: string[];
  cache_size?: number;
  deny?: string[];
  empty_user_agent_is_bot?: boolean;
  patterns?: string[];
}

export interface KrakendSecurityCors {
  allow_credentials?: boolean;
  allow_headers?: string[];
  allow_methods?: KrakendHttpMethod[];
  allow_origins?: string[];
  allow_private_network?: boolean;
  debug?: boolean;
  expose_headers?: string[];
  max_age?: string[];
  options_passthrough?: boolean;
  options_success_status?: number;
}

export interface KrakendSecurityHttp {
  allowed_hosts?: string[];
  allowed_hosts_are_regex?: boolean;
  browser_xss_filter?: boolean;
  content_security_policy?: string;
  content_type_nosniff?: boolean;
  custom_frame_options_value?: string;
  force_sts_header?: boolean;
  frame_deny?: boolean;
  host_proxy_headers?: string[];
  hpkp_public_key?: string;
  is_development?: boolean;
  referrer_policy?: string;
  ssl_host?: string;
  ssl_proxy_headers?: Record<string, string>[];
  ssl_redirect?: boolean;
  sts_include_subdomains?: boolean;
  sts_seconds?: number;
}

export interface KrakendValidationCel {
  check_expr: string;
}

export interface KrakendEndpointExtraConfig {
  'auth/signer'?: KrakendAuthSigner;
  'auth/validator'?: KrakendAuthValidator;
  'modifier/lua-endpoint'?: KrakendModifierLua;
  'modifier/lua-proxy'?: KrakendModifierLua;
  proxy?: {
    combiner?: string;
    flatmap_filter?: string[];
    sequential?: boolean;
    sequential_propagated_params?: string[];
    static?: {
      data: object;
      strategy: 'always' | 'success' | 'complete' | 'errored' | 'incomplete';
    };
  };
  'qos/ratelimit/router'?: KrakendQosRateLimitRouter;
  'security/bot-detector'?: KrakendSecurityBotDetector;
  'security/cors'?: KrakendSecurityCors;
  'security/http'?: KrakendSecurityHttp;
  'validation/cel'?: KrakendValidationCel;
  'validation/json-schema'?: object;
}

export interface KrakendAsyncAmqp {
  auto_ack?: boolean;
  delete?: boolean;
  durable?: boolean;
  exchange: string;
  exclusive?: boolean;
  host: string;
  nack_discard?: boolean;
  name: string;
  no_local?: boolean;
  no_wait?: boolean;
  prefetch_count?: number;
  prefetch_size?: number;
}

export interface KrakendAsyncAgent {
  backend: KrakendBackend;
  connection?: {
    backoff_strategy?:
      | 'linear'
      | 'linear-jitter'
      | 'exponential'
      | 'exponential-jitter'
      | 'fallback';
    health_interval?: string;
    max_retries?: number;
  };
  consumer: {
    max_rate?: number;
    timeout?: string;
    topic: string;
    workers?: number;
  };
  encoding?: 'json' | 'safejson' | 'xml' | 'rss' | 'string' | 'no-op';
  extra_config: {
    'async/amqp': KrakendAsyncAmqp;
  };
  name: string;
}

export interface KrakendAiLlm {
  anthropic?: object; // ToDo
  gemini?: object; // ToDo
  mistral?: object; // ToDo
  openai?: object; // ToDo
}

export interface KrakendAuthAwsSig4 {
  assume_role_arn?: string;
  debug?: boolean;
  region: string;
  service: string;
  sts_region?: string;
}

export interface KrakendAuthClientCredentials {
  client_id: string;
  client_secret: string;
  endpoint_params?: object;
  scopes?: string;
  token_url: string;
}

export interface KrakendAmqpConsumer {
  auto_ack?: boolean;
  backoff_strategy?:
    | 'linear'
    | 'linear-jitter'
    | 'exponential'
    | 'exponential-jitter'
    | 'fallback';
  delete?: boolean;
  durable?: boolean;
  exchange: string;
  exclusive?: boolean;
  max_retries?: number;
  nack_discard?: boolean;
  name: string;
  no_local?: boolean;
  no_wait?: boolean;
  prefetch_count?: number;
  routing_key: string[];
}

export type KrakendAmqpProducer = Omit<
  KrakendAmqpConsumer,
  'auto_ack' | 'nack_discard' | 'prefertch_count'
> & {
  exp_key?: string;
  immediate?: boolean;
  mandatory?: boolean;
  msg_id_key?: string;
  priority_key?: string;
  reply_to_key?: string;
  routing_key: string;
  static_routing_key?: string;
};

export interface KrakendBackendConditional {
  name?: string;
  strategy: 'header' | 'policy' | 'fallback';
  value?: string;
}

export interface KrakendBackendGraphQl {
  type?: 'query' | 'mutation';
  operationName?: string;
  query?: string;
  query_path?: string;
  variables?: object;
}

export interface KrakendBackendPubSubPublisher {
  topic_url: string;
}

export interface KrakendBackendPubSubSubscriber {
  subscription_url: string;
}

export interface KrakendProxyFlatmapItem {
  type: 'move' | 'del' | 'append';
  args: string[];
}

export interface KrakendProxy {
  flatmap_filter?: KrakendProxyFlatmapItem[];
  shadow?: boolean;
}

export interface KrakendCurcuitBreaker {
  interval: number;
  log_status_change?: boolean;
  max_errors: number;
  name?: string;
  timeout: number;
}

export interface KrakendRateLimitProxy {
  capacity: number;
  every?: string;
  max_rate: number;
}

export interface KrakendTelemetryLogging {
  format?: string;
  access_log_custom_format?: string;
  access_log_format?:
    | 'default'
    | 'httpdCommon'
    | 'httpdCombine'
    | 'json'
    | 'custom';
  access_log_missing_key_marker?: string;
  custom_format?: string;
  level: 'DEBUG' | 'INFO' | 'WARNING' | 'ERROR' | 'CRITICAL';
  prefix?: string;
  stdout?: boolean;
  syslog?: boolean;
  syslog_facility?:
    | 'local0'
    | 'local1'
    | 'local2'
    | 'local3'
    | 'local4'
    | 'local5'
    | 'local6'
    | 'local7';
}

export interface KrakendWorkflowExtraConfig {
  'modifier/jmespath'?: object; // ToDo
  'modifier/lua-proxy'?: object; // ToDo
  proxy?: {
    combiner?: string;
    flatmap_filter?: KrakendProxyFlatmapItem[];
    sequential?: boolean;
    sequential_propagated_params?: string[];
    static?: {
      data: object;
      strategy: 'always' | 'success' | 'complete' | 'errored' | 'incomplete';
    };
  };
  'validation/cel'?: KrakendValidationCel;
  'validation/json-schema'?: object;
}

export interface KrakendWorkflow {
  backend: KrakendBackend;
  concurrent_calls?: number;
  endpoint: string;
  extra_config?: KrakendWorkflowExtraConfig;
  ignore_errors?: boolean;
  output_encoding?:
    | 'json'
    | 'json-collection'
    | 'yaml'
    | 'fast-json'
    | 'xml'
    | 'negotiate'
    | 'string'
    | 'no-op';
  timeout?: string;
}

export interface KrakendBackendExtraConfig {
  'ai/llm'?: KrakendAiLlm;
  'auth/aws-sigv4'?: KrakendAuthAwsSig4;
  'auth/client-credentials'?: KrakendAuthClientCredentials;
  'backend/amqp/consumer'?: KrakendAmqpConsumer;
  'backend/amqp/producer'?: KrakendAmqpProducer;
  'backend/conditional'?: KrakendBackendConditional;
  'backend/graphql'?: KrakendBackendGraphQl;
  'backend/http'?: {
    return_error_code?: boolean;
    return_error_details?: string;
  };
  'backend/lambda'?: object; // ToDo
  'backend/pubsub/publisher'?: KrakendBackendPubSubPublisher;
  'backend/pubsub/subscriber'?: KrakendBackendPubSubSubscriber;
  'backend/soap'?: object; // ToDo
  'backend/static-filesystem'?: object; // ToDo
  'modifier/lua-backend'?: KrakendModifierLua;
  'modifier/martian'?: object; // ToDo
  'plugin/http-client'?: { name: string };
  proxy?: KrakendProxy;
  'qos/circuit-breaker'?: KrakendCurcuitBreaker;
  'qos/http-cache'?: {
    max_items?: number;
    max_size?: number;
    shared?: boolean;
  };
  'qos/ratelimit/proxy'?: KrakendRateLimitProxy;
  'telemetry/logging'?: KrakendTelemetryLogging;
  'telemetry/opentelemetry'?: object; // ToDo
  'validation/cel'?: KrakendValidationCel;
  workflow?: KrakendWorkflow;
}

export interface KrakendBackend {
  allow?: string[];
  deny?: string[];
  disable_host_sanitize?: boolean;
  encoding?:
    | 'json'
    | 'safejson'
    | 'fast-json'
    | 'xml'
    | 'rss'
    | 'string'
    | 'no-op'
    | 'yaml';
  extra_config?: KrakendBackendExtraConfig;
  group?: string;
  host?: string[];
  input_headers?: string[];
  input_query_strings?: string[];
  is_collection?: boolean;
  mapping?: Record<string, string>;
  method?: KrakendHttpMethod;
  sd?: 'static' | 'dns' | 'dns-shared';
  sd_scheme?: string;
  target?: string;
  url_pattern: string;
}

export interface KrakendEndpoint {
  backend: KrakendBackend[];
  cache_ttl?: string;
  concurrent_calls?: number;
  endpoint: string;
  extra_config?: KrakendEndpointExtraConfig;
  input_headers?: string[];
  input_query_strings?: string[];
  method: KrakendHttpMethod;
  output_encoding?: KrakendEncoding;
  timeout?: string;
}
