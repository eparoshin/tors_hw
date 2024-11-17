#include "discovery.h"

#include <arpa/inet.h>

#include <string>
#include <iostream>
#include <sstream>
#include <string>

int main(int argc, char** argv) {
    std::vector<NClient::TPoint> points;
    std::string line;
    if (argc != 3) {
        std::cerr << "incorrect args";
        return 1;
    }
    while (std::getline(std::cin, line)) {
        std::stringstream ss(line);
        double x, y;
        ss >> x >> y;
        points.emplace_back(x, y);
    }

    in_addr broadcastAddr;
    broadcastAddr.s_addr = inet_addr(argv[1]);

    uint16_t port = std::stoi(argv[2]);

    NClient::TDiscovery discovery(broadcastAddr, port);
    discovery.UpdateList(1000);

    std::cout << discovery.Calc(points) << std::endl;
}
