// src/api/index.ts
import { BackendConfig } from "./types";
import { Health } from "./routes/health";
import { Group } from "./routes/group";

export class BackendClient {
  public health: Health;
  public group: Group;

  constructor(config: BackendConfig) {
    this.health = new Health(config)
    this.group = new Group(config)
  }
}