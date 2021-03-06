-- // create Table
-------------------------------------------------------------------------------
CREATE TABLE IF NOT EXISTS azmonitor.application (
  applicationID SERIAL,
  subscription_id varchar,
  name varchar,
  tenant_id varchar,
  grant_type varchar,
  client_id varchar,
  client_secret varchar,
  lastmodified TIMESTAMP WITH TIME ZONE DEFAULT CURRENT_TIMESTAMP,
  PRIMARY KEY (applicationID)
);


CREATE TABLE IF NOT EXISTS azmonitor.virtualmachine (
  id UUID UNIQUE NOT NULL DEFAULT uuid_generate_v4(),
  resourceID  varchar,
  resourceGroup varchar,
  serviceName varchar,
  cost varchar,
  resourceType varchar,
  resourceLocation varchar,
  consumptionType varchar,
  meter varchar,
  cpuUtilization varchar,
  availableMemory varchar,
  diskLatency varchar,
  diskIOPs varchar,
  diskBytesPerSec varchar,
  networkSentRate varchar,
  networkReceivedRate varchar,
  dateCreated TIMESTAMP,
  lastUpdated  TIMESTAMP,
  reportStartDate varchar,
  reportEndDate varchar,
  data jsonb,
  PRIMARY KEY (id)
);


CREATE TABLE IF NOT EXISTS azmonitor.storageaccount (
  id UUID UNIQUE NOT NULL DEFAULT uuid_generate_v4(),
  resourceID  varchar,
  resourceGroup varchar,
  serviceName varchar,
  cost varchar,
  resourceType varchar,
  resourceLocation varchar,
  consumptionType varchar,
  meter varchar,
  availability varchar,
  totalTransactions varchar,
  e2ELatency varchar, 
  serverLantency varchar,
  failures varchar,
  capacity varchar,
  dateCreated TIMESTAMP,
  lastUpdated  TIMESTAMP,
  reportStartDate varchar,
  reportEndDate varchar,
  data jsonb,
  PRIMARY KEY (id)
); 
