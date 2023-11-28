import React, { useState } from "react";
import { Button, FieldSet, Modal } from "@grafana/ui";

export const DevSandbox = () => {
    const [modalIsOpen, setModalIsOpen] = useState(false);

    return (
        <FieldSet label="Development Sandbox">
            <Button onClick={() => setModalIsOpen(true)}>Open development sandbox</Button>
                <Modal title="Development Sandbox" isOpen={modalIsOpen}>
                    <Button variant="primary" onClick={() => setModalIsOpen(false)}>
                    Close
                    </Button>
                </Modal>
        </FieldSet>
    );
};
