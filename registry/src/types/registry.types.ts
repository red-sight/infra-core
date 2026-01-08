export interface ServiceInfo {
  name: string;
  ip: string;
  port: number;
  version: string;
  oasEndpoint?: string;
  healthEndpoint?: string;
}
