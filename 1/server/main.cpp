#include "options.h"
#include "discovery_server.h"
#include "worker_server.h"

int main(int argc, char** argv) {
    using namespace NServer;
    auto config = TConfig::Parse(argc, argv);

    TDiscoveryServer dServer(config.ListenPort);
    dServer.Start();
    TWorkerServer wServer(config.ListenPort);
    wServer.Start();

    dServer.Join();
    wServer.Join();
}
