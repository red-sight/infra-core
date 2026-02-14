export interface ServiceInfo {
  name: string;
  ip: string;
  port: number;
  version: string;
  oasEndpoint?: string;
  healthEndpoint?: string;
}

export interface IDockerEvent {
  Actor?: {
    Attributes?: {
      name?: string;
    };
  };
  Action?: string;
}
