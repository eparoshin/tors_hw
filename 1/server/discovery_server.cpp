#include "discovery_server.h"
#include "defer.h"

#include <sys/socket.h>
#include <netinet/in.h>
#include <arpa/inet.h>
#include <unistd.h>

#include <iostream>
#include <array>
#include <string_view>
#include <cstring>

namespace NServer {
    namespace {
        constexpr size_t kBufSize = 1024;
        void RunServer(const uint16_t port) {
            std::array<char, kBufSize> buff;
            int sockfd;
            sockaddr_in servaddr, cliaddr;
            if ((sockfd = socket(AF_INET, SOCK_DGRAM, 0)) < 0) {
                perror("Socket creation failed");
                exit(EXIT_FAILURE);
            }

            NUtil::TDefer closeSocket([sockfd]() { close(sockfd); });

            std::memset(&servaddr, 0, sizeof(servaddr));
            servaddr.sin_family = AF_INET;
            servaddr.sin_addr.s_addr = INADDR_ANY;
            servaddr.sin_port = htons(port);

            if (bind(sockfd, (const struct sockaddr*)&servaddr, sizeof(servaddr)) < 0) {
                perror("Bind failed");
                exit(EXIT_FAILURE);
            }

            std::cout << "Server listening on port " << port << "...\n";

            while (true) {
                socklen_t len = sizeof(cliaddr);

                auto n = recvfrom(sockfd, buff.data(), buff.size(), 0, (struct sockaddr*)&cliaddr, &len);
                if (n < 0) {
                    perror("recvfrom error");
                    continue;
                }

                auto msg = std::string_view(buff.data(), n);

                std::cout << "Received message: " << msg
                          << " from " << inet_ntoa(cliaddr.sin_addr) << "\n";

                if (msg.starts_with("END")) {
                    return;
                }

                constexpr std::string_view response("PONG");

                auto sent = sendto(sockfd, response.data(), response.size(), 0,
                                      (const struct sockaddr*)&cliaddr, len);
                if (sent < 0) {
                    perror("sendto error");
                } else {
                    std::cout << "Sent response to " << inet_ntoa(cliaddr.sin_addr) << "\n";
                }
            }

        }
    }

    TDiscoveryServer::TDiscoveryServer(uint16_t port)
    : TServer(port) {
    }

    void TDiscoveryServer::Start() {
        TServer::Start(RunServer);
    }


}
