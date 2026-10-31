import { simulation, scenario, constantUsersPerSec, rampUsersPerSec, global } from "@gatling.io/core";
import { http, status, bodyString } from "@gatling.io/http";

const baseUrl = process.env.NESTGO_BASE_URL || "http://127.0.0.1:18080";
const rate = Number(process.env.NESTGO_RATE || "10");
const duration = Number(process.env.NESTGO_DURATION || "30");
const warmup = Number(process.env.NESTGO_WARMUP || "10");

export default simulation((setUp) => {
  const protocol = http.baseUrl(baseUrl).acceptHeader("application/json");
  const read = scenario("GET users list")
    .exec(http("GET /users/").get("/users/")
      .check(status().is(200), bodyString().exists()));
  setUp(read.injectOpen(
    rampUsersPerSec(1).to(rate).during(warmup),
    constantUsersPerSec(rate).during(duration)
  )).protocols(protocol)
    .assertions(global().failedRequests().count().is(0));
});
