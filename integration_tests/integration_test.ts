import { expect } from 'chai';
import camelCase from 'lodash.camelcase';
import { pathOr } from 'ramda';
import { CounterService } from "./service.pb";

function getFieldName(name: string) {
  const useCamelCase = pathOr(false, ['__karma__', 'config', 'useProtoNames'], window) === false
  return useCamelCase ? camelCase(name) : name
}

function getField(obj: {[key: string]: any}, name: string) {
  return obj[getFieldName(name)]
}

describe("test grpc-gateway-ts communication", () => {
  it("unary request", async () => {
    const result = await CounterService.Increment({ counter: 199 }, { pathPrefix: "http://localhost:8081" })

    expect(result.result).to.equal(200)
  })

  it('streaming request', async () => {
    const response = [] as number[]
    await CounterService.StreamingIncrements({ counter: 1 }, (resp) => response.push(resp.result), { pathPrefix: "http://localhost:8081" })

    expect(response).to.deep.equal([2, 3, 4, 5, 6])
  })

  it('http get check request', async () => {
    const result = await CounterService.HTTPGet({ [getFieldName('num_to_increase')]: 10 }, { pathPrefix: "http://localhost:8081" })
    expect(result.result).to.equal(11)
  })

  it('http post body check request with nested body path', async () => {
    const result = await CounterService.HTTPPostWithNestedBodyPath({ a: 10, req: { b: 15 } }, { pathPrefix: "http://localhost:8081" })
    expect(getField(result, 'post_result')).to.equal(25)
  })

  it('http post body check request with star in path', async () => {
    const result = await CounterService.HTTPPostWithStarBodyPath({ a: 10, req: { b: 15 }, c: 23 }, { pathPrefix: "http://localhost:8081" })
    expect(getField(result, 'post_result')).to.equal(48)
  })

  it('able to communicate with external message reference without package defined', async () => {
    const result = await CounterService.ExternalMessage({ content: "hello" }, { pathPrefix: "http://localhost:8081" })
    expect(getField(result, 'result')).to.equal("hello!!")
  })

  it('http patch request with star in path', async () => {
    const result = await CounterService.HTTPPatch({ a: 10, c: 23 }, { pathPrefix: "http://localhost:8081" })
    expect(getField(result, 'patch_result')).to.equal(33)
  })

  it('http delete check request', async () => {
    const result = await CounterService.HTTPDelete({ a: 10 }, { pathPrefix: "http://localhost:8081" })
    expect(result).to.be.empty
  })
    
  it('http get request with url search parameters', async () => {
    const result = await CounterService.HTTPGetWithURLSearchParams({ a: 10, [getFieldName('post_req')]: { b: 0 }, c: [23, 25], [getFieldName('ext_msg')]: { d: 12 } }, { pathPrefix: "http://localhost:8081" })
    expect(getField(result, 'url_search_params_result')).to.equal(70)
  })

  it('http get request with zero value url search parameters', async () => {
    const result = await CounterService.HTTPGetWithZeroValueURLSearchParams({ a: "A", b: "", [getFieldName('zero_value_msg')]: { c: 1, d: [1, 0, 2], e: false } }, { pathPrefix: "http://localhost:8081" })
    expect(result).to.deep.equal({ a: "A", b: "hello", [getFieldName('zero_value_msg')]: { c: 2, d: [2, 1, 3], e: true } })
  })
})
