#include "discovery.h"

#include <string>
#include <iostream>
#include <sstream>

int main() {
    std::vector<NClient::TPoint> points;
    std::string line;
    while (std::getline(std::cin, line)) {
        std::stringstream ss(line);
        double x, y;
        ss >> x >> y;
        points.emplace_back(x, y);
    }

    NClient::TDiscovery discovery(10);
    discovery.UpdateList();

    std::cout << discovery.Calc(points) << std::endl;
}
