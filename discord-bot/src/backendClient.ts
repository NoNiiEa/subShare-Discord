import axios, { AxiosInstance } from "axios";

export interface HealthResponse {
    status: string;
}

export class BackendClient {
    private http: AxiosInstance;
    
    constructor(baseURL: string) {
        this.http = axios.create({
            baseURL,
            timeout: 5000,
        });
    }

    async health(): Promise<HealthResponse> {
        const res = await this.http.get<HealthResponse>("/health");
        return res.data;
    }
}