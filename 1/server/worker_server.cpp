#include "worker_server.h"
#include "defer.h"
#include "parser.h"
#include "calc.h"


#include <unistd.h>
#include <sys/types.h>
#include <sys/socket.h>
#include <netinet/in.h>
#include <arpa/inet.h>

#include <thread>
#include <iostream>
#include <array>
#include <vector>
#include <span>
#include <bit>

namespace NServer {
    namespace {
        constexpr auto HostToInet(std::integral auto i) {
            static_assert(std::endian::native == std::endian::big || std::endian::native == std::endian::little);
            if constexpr (std::endian::native == std::endian::big) {
                return i;
            } else {
                return std::byteswap(i);
            }
        }

        constexpr size_t kBuffSize = 1024;
        void HandleClient(int clientSocket) {
            NUtil::TDefer closeSock([clientSocket]() { close(clientSocket); });
            ssize_t bytes_read;
            std::array<char, kBuffSize> buff;
            std::vector<char> input;
            while ((bytes_read = read(clientSocket, buff.data(), buff.size())) > 0) {
                input.insert(input.end(), buff.begin(), buff.begin() + bytes_read);
            }

            double result = CalculateSquare(Parse(input));

            std::span<char> out(reinterpret_cast<char*>(&result), sizeof(result));
            while (!out.empty()) {
                auto n = write(clientSocket, out.data(), out.size());
                if (n <= 0) {
                    perror("write");
                    return;
                }
                out = out.last(out.size() - n);
            }

        }

        void RunServer(uint16_t port) {
            int serverfd;
            sockaddr_in address;
            int opt = 1;

            if ((serverfd = socket(AF_INET, SOCK_STREAM, 0)) == -1) {
                perror("socket failed");
                exit(EXIT_FAILURE);
            }

            NUtil::TDefer closeSocket([serverfd]() { close(serverfd); });

            if (setsockopt(serverfd, SOL_SOCKET, SO_REUSEADDR, &opt, sizeof(opt))) {
                perror("setsockopt");
                exit(EXIT_FAILURE);
            }

            address.sin_family = AF_INET;
            address.sin_addr.s_addr = INADDR_ANY;
            address.sin_port = htons(port);

            if (bind(serverfd, (struct sockaddr *)&address, sizeof(address)) < 0) {
                perror("bind failed");
                exit(EXIT_FAILURE);
            }

            if (listen(serverfd, 5) < 0) {
                perror("listen");
                exit(EXIT_FAILURE);
            }

            std::cout << "Server listening on port " << port << std::endl;

            while (true) {
                int newSocket;
                sockaddr_in clientAddress;
                socklen_t clientAddrlen = sizeof(clientAddress);

                if ((newSocket = accept(serverfd, (sockaddr *)&clientAddress, &clientAddrlen)) < 0) {
                    perror("accept");
                    continue;
                }

                std::cout << "New connection from "
                          << inet_ntoa(clientAddress.sin_addr) << ":"
                          << ntohs(clientAddress.sin_port) << std::endl;

                std::thread t(HandleClient, newSocket);
                t.detach();
            }
        }
    }


    TWorkerServer::TWorkerServer(uint16_t port)
    : TServer(port) {
    }

    void TWorkerServer::Start() {
        TServer::Start(RunServer);
    }

}
