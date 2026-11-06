// path: backend/gateway/src/grpc.ts

import * as grpc from '@grpc/grpc-js';
import * as protoLoader from '@grpc/proto-loader';
import path from 'path';
import { getGrpcClientCredentials } from './mtls';

const PROTO_ROOT = process.env.PROTO_ROOT || path.resolve(__dirname, '../../../proto');

function getNested(obj: any, keyPath: string): any {
  return keyPath.split('.').reduce((prev, curr) => (prev ? prev[curr] : undefined), obj);
}

function loadServiceClient(protoFile: string, packageName: string, serviceName: string): any {
  const filePath = path.join(PROTO_ROOT, protoFile);
  const packageDefinition = protoLoader.loadSync(filePath, {
    keepCase: true,
    longs: String,
    enums: String,
    defaults: true,
    oneofs: true,
    includeDirs: [PROTO_ROOT],
  });

  const packageDescriptor = grpc.loadPackageDefinition(packageDefinition);
  const pkg = getNested(packageDescriptor, packageName);
  if (!pkg || !pkg[serviceName]) {
    throw new Error(`Failed to find service ${serviceName} in package ${packageName} at ${filePath}`);
  }
  return pkg[serviceName];
}

export function createUserClient(address = process.env.USER_SERVICE_ADDR || 'localhost:50051') {
  const ServiceConstructor = loadServiceClient('user/v1/user.proto', 'user.v1', 'UserService');
  return new ServiceConstructor(address, getGrpcClientCredentials('user'));
}

export function createProductClient(address = process.env.PRODUCT_SERVICE_ADDR || 'localhost:50052') {
  const ServiceConstructor = loadServiceClient('product/v1/product.proto', 'product.v1', 'ProductService');
  return new ServiceConstructor(address, getGrpcClientCredentials('product'));
}

export function createOrderClient(address = process.env.ORDER_SERVICE_ADDR || 'localhost:50053') {
  const ServiceConstructor = loadServiceClient('order/v1/order.proto', 'order.v1', 'OrderService');
  return new ServiceConstructor(address, getGrpcClientCredentials('order'));
}

export function createPaymentClient(address = process.env.PAYMENT_SERVICE_ADDR || 'localhost:50054') {
  const ServiceConstructor = loadServiceClient('payment/v1/payment.proto', 'payment.v1', 'PaymentService');
  return new ServiceConstructor(address, getGrpcClientCredentials('payment'));
}

export function invokeGrpc<TReq, TRes>(clientMethod: (req: TReq, cb: (err: any, res: TRes) => void) => void, request: TReq): Promise<TRes> {
  return new Promise((resolve, reject) => {
    clientMethod(request, (err: any, response: TRes) => {
      if (err) {
        return reject(err);
      }
      resolve(response);
    });
  });
}
