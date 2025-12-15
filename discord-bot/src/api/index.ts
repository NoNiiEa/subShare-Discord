// src/api/index.ts
import { BackendConfig } from "./types.js";
import { Health } from "./routes/health.js";
import { Group } from "./routes/group.js";
import { Bill } from "./routes/bill.js";

export class BackendClient {
  public health: Health;
  public group: Group;
  public bill: Bill;

  constructor(config: BackendConfig) {
    this.health = new Health(config)
    this.group = new Group(config)
    this.bill = new Bill(config)
  }
}