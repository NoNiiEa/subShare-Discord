import axios from "axios";
export class BackendClient {
    http;
    constructor(baseURL) {
        this.http = axios.create({
            baseURL,
            timeout: 5000,
        });
    }
    async health() {
        const res = await this.http.get("/health");
        return res.data;
    }
}
